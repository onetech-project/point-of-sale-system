-- Migration 000034: Remove privacy policy v1.0.0
DELETE FROM privacy_policies WHERE version = '1.0.0';