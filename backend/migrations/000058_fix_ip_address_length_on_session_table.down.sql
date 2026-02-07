ALTER TABLE consent_records
ALTER COLUMN ip_address TYPE INET USING ip_address::inet;