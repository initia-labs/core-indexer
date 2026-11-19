package repositories

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gocloud.dev/blob"
	"gocloud.dev/blob/driver"
	"gocloud.dev/blob/memblob"
	"gocloud.dev/gcerrors"
)

type listErrorBucket struct {
	driver.Bucket
	err   error
	calls int
}

func (b *listErrorBucket) ListPaged(ctx context.Context, _ *driver.ListOptions) (*driver.ListPage, error) {
	b.calls++
	if b.calls == 1 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return nil, b.err
	}
	// End an incorrect retry so the regression fails without hanging.
	return &driver.ListPage{}, nil
}

func (b *listErrorBucket) ErrorCode(error) gcerrors.ErrorCode { return gcerrors.Unknown }

func TestGetTxByHashListError(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "permission", err: errors.New("storage.objects.list denied")},
		{name: "authentication", err: errors.New("oauth2: invalid_grant")},
	} {
		for _, candidate := range []bool{false, true} {
			name := tc.name
			if candidate {
				name += "/earlier_candidate"
			}
			t.Run(name, func(t *testing.T) {
				var buckets []*blob.Bucket
				if candidate {
					buckets = append(buckets, newTxTestBucket(t, map[string]string{
						"ABC/10": `{"tx_response":{"height":"10"}}`,
					}))
				}
				failed := &listErrorBucket{err: tc.err}
				buckets = append(buckets, blob.NewBucket(failed))

				tx, err := NewTxRepository(nil, buckets, 0).GetTxByHash(context.Background(), "abc")

				require.ErrorIs(t, err, tc.err)
				require.Nil(t, tx, "a failed search must not return a potentially stale candidate")
				require.Equal(t, 1, failed.calls, "a list failure must not be retried")
			})
		}
	}
}

func TestGetTxByHashContextError(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, cancel := context.WithDeadline(context.Background(), time.Time{})
	defer cancel()

	for _, tc := range []struct {
		name string
		ctx  context.Context
		err  error
	}{
		{name: "canceled", ctx: canceled, err: context.Canceled},
		{name: "deadline_exceeded", ctx: expired, err: context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			failed := &listErrorBucket{}
			repo := NewTxRepository(nil, []*blob.Bucket{blob.NewBucket(failed)}, 0)

			tx, err := repo.GetTxByHash(tc.ctx, "abc")

			require.ErrorIs(t, err, tc.err)
			require.Nil(t, tx)
			require.Equal(t, 1, failed.calls, "a completed context must not be retried")
		})
	}
}

func TestGetTxByHashLatestAcrossBuckets(t *testing.T) {
	older := newTxTestBucket(t, map[string]string{
		"ABC/9":  `{"tx_response":{"height":"9"}}`,
		"ABC/20": `{"tx_response":{"height":"20"}}`,
	})
	newer := newTxTestBucket(t, map[string]string{
		"ABC/15":    `{"tx_response":{"height":"15"}}`,
		"ABC/100":   `{"tx_response":{"height":"100"}}`,
		"OTHER/200": `{"tx_response":{"height":"200"}}`,
	})
	for _, tc := range []struct {
		name    string
		buckets []*blob.Bucket
	}{
		{name: "newest_first", buckets: []*blob.Bucket{newer, older}},
		{name: "newest_last", buckets: []*blob.Bucket{older, newer}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := NewTxRepository(nil, tc.buckets, 0).GetTxByHash(context.Background(), "abc")
			require.NoError(t, err)
			require.Equal(t, "100", tx.TxResponse.Height)
		})
	}
}

func newTxTestBucket(t *testing.T, objects map[string]string) *blob.Bucket {
	t.Helper()
	bucket := memblob.OpenBucket(nil)
	t.Cleanup(func() { require.NoError(t, bucket.Close()) })
	for key, data := range objects {
		require.NoError(t, bucket.WriteAll(context.Background(), key, []byte(data), nil))
	}
	return bucket
}
