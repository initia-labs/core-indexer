package main

import (
	"os"
	"strconv"
)

type Config struct {
	ID string

	PubSubProjectID     string
	PubSubEndpoint      string
	LCDTXResponseBucket string

	TxResponseSubscription string

	Chain       string
	Environment string

	ClaimCheckBucket string

	SentryDSN               string
	CommitSHA               string
	SentryTraceSampleRate   float64
	SentryProfileSampleRate float64
}

func getConfig() *Config {
	// all env vars are required, panic if doesn't set
	if os.Getenv("ID") == "" {
		logger.Fatal().Msg("ID is required")
	}
	if os.Getenv("PUBSUB_PROJECT_ID") == "" {
		logger.Fatal().Msg("PUBSUB_PROJECT_ID is required")
	}
	if os.Getenv("LCD_TX_RESPONSE_BUCKET") == "" {
		logger.Fatal().Msg("LCD_TX_RESPONSE_BUCKET is required")
	}
	if os.Getenv("TX_RESPONSE_SUBSCRIPTION") == "" {
		logger.Fatal().Msg("TX_RESPONSE_SUBSCRIPTION is required")
	}
	if os.Getenv("CHAIN") == "" {
		logger.Fatal().Msg("CHAIN is required")
	}
	if os.Getenv("ENVIRONMENT") == "" {
		logger.Fatal().Msg("ENVIRONMENT is required")
	}
	if os.Getenv("CLAIM_CHECK_BUCKET") == "" {
		logger.Fatal().Msg("CLAIM_CHECK_BUCKET is required")
	}
	var traceSampleRate float64
	var err error
	if os.Getenv("SENTRY_TRACE_SAMPLE_RATE") == "" {
		traceSampleRate = 0.01
	} else {
		traceSampleRate, err = strconv.ParseFloat(os.Getenv("SENTRY_TRACE_SAMPLE_RATE"), 64)
		if err != nil {
			logger.Error().Msgf("Error parsing SENTRY_TRACE_SAMPLE_RATE: %v", err)
			traceSampleRate = 0.01
		}
	}

	var profileSampleRate float64
	if os.Getenv("SENTRY_PROFILE_SAMPLE_RATE") == "" {
		profileSampleRate = 0.01
	} else {
		profileSampleRate, err = strconv.ParseFloat(os.Getenv("SENTRY_PROFILE_SAMPLE_RATE"), 64)
		if err != nil {
			logger.Error().Msgf("Error parsing SENTRY_PROFILE_SAMPLE_RATE: %v", err)
			profileSampleRate = 0.01
		}
	}

	return &Config{
		ID:                      os.Getenv("ID"),
		PubSubProjectID:         os.Getenv("PUBSUB_PROJECT_ID"),
		PubSubEndpoint:          os.Getenv("PUBSUB_ENDPOINT"),
		LCDTXResponseBucket:     os.Getenv("LCD_TX_RESPONSE_BUCKET"),
		TxResponseSubscription:  os.Getenv("TX_RESPONSE_SUBSCRIPTION"),
		Chain:                   os.Getenv("CHAIN"),
		Environment:             os.Getenv("ENVIRONMENT"),
		ClaimCheckBucket:        os.Getenv("CLAIM_CHECK_BUCKET"),
		SentryDSN:               os.Getenv("SENTRY_DSN"),
		CommitSHA:               os.Getenv("COMMIT_SHA"),
		SentryTraceSampleRate:   traceSampleRate,
		SentryProfileSampleRate: profileSampleRate,
	}
}
