package txparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMessageDicts(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		want    []map[string]any
		wantErr bool
	}{
		{
			name: "valid transaction with single message",
			input: map[string]any{
				"tx": map[string]any{
					"body": map[string]any{
						"messages": []any{
							map[string]any{
								"type": "test/Message",
								"value": map[string]any{
									"field1": "value1",
								},
							},
						},
					},
				},
			},
			want: []map[string]any{
				{
					"type": "test/Message",
					"value": map[string]any{
						"field1": "value1",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "nil input",
			input:   nil,
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing tx field",
			input: map[string]any{
				"not_tx": map[string]any{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing body field",
			input: map[string]any{
				"tx": map[string]any{
					"not_body": map[string]any{},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing messages field",
			input: map[string]any{
				"tx": map[string]any{
					"body": map[string]any{
						"not_messages": []any{},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "messages not a slice",
			input: map[string]any{
				"tx": map[string]any{
					"body": map[string]any{
						"messages": "not a slice",
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "message not a map",
			input: map[string]any{
				"tx": map[string]any{
					"body": map[string]any{
						"messages": []any{"not a map"},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "multiple valid messages",
			input: map[string]any{
				"tx": map[string]any{
					"body": map[string]any{
						"messages": []any{
							map[string]any{"foo": "bar"},
							map[string]any{"baz": 123},
						},
					},
				},
			},
			want: []map[string]any{
				{"foo": "bar"},
				{"baz": 123},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseMessages(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}
