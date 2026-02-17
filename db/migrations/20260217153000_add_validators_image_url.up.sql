-- Add identity_image column to validators table to store pre-fetched Keybase image data (base64-encoded)
-- This avoids on-demand API calls to Keybase during API requests
ALTER TABLE validators ADD COLUMN identity_image text DEFAULT '';
