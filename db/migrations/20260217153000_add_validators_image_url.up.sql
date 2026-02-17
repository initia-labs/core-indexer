-- Add image_url column to validators table to store pre-fetched Keybase images
-- This avoids on-demand API calls to Keybase during API requests
ALTER TABLE validators ADD COLUMN image_url varchar DEFAULT '';
