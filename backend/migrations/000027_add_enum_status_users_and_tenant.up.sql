ALTER TABLE users ALTER COLUMN status SET DEFAULT 'inactive';

ALTER TABLE users
DROP CONSTRAINT IF EXISTS users_status_check,
ADD CONSTRAINT users_status_check CHECK (
    status IN (
        'active',
        'suspended',
        'deleted',
        'inactive'
    )
);

ALTER TABLE tenants ALTER COLUMN status SET DEFAULT 'inactive';

ALTER TABLE tenants
DROP CONSTRAINT IF EXISTS tenants_status_check,
ADD CONSTRAINT tenants_status_check CHECK (
    status IN (
        'active',
        'suspended',
        'deleted',
        'inactive'
    )
);