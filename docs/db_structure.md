categories

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | uuid_generate_v4() |
| tenant_id | uuid |     |     | NO  |     |
| name | varchar(100) |     |     | NO  |     |
| display_order | integer |     |     | NO  | 0   |
| created_at | timestamp without time zone |     |     | NO  | now() |
| updated_at | timestamp without time zone |     |     | NO  | now() |

delivery_addresses

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| order_id | uuid |     |     | NO  |     |
| address_text | text |     |     | NO  |     |
| latitude | numeric(10,8) |     |     | YES |     |
| longitude | numeric(11,8) |     |     | YES |     |
| geocoded_address | text |     |     | YES |     |
| place_id | varchar(255) |     |     | YES |     |
| is_serviceable | boolean | False if address outside service area |     | NO  | false |
| service_area_zone | varchar(100) |     |     | YES |     |
| calculated_delivery_fee | integer |     |     | YES |     |
| distance_km | numeric(6,2) | Distance from tenant location (Haversine) |     | YES |     |
| geocoded_at | timestamp without time zone |     |     | YES |     |
| created_at | timestamp without time zone |     |     | NO  | now() |

event_records

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| event_id | uuid |     | YES | NO  |     |
| order_id | uuid |     |     | YES |     |
| tenant_id | uuid |     |     | NO  |     |
| event_type | text |     |     | NO  |     |
| processed_at | timestamp with time zone |     |     | NO  | now() |
| metadata | jsonb |     |     | YES |     |
| created_at | timestamp with time zone |     |     | NO  | now() |

guest_orders

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| order_reference | varchar(20) | Human-readable order reference: GO-XXXXXX |     | NO  |     |
| tenant_id | uuid |     |     | NO  |     |
| status | varchar(20) | Order lifecycle: PENDING → PAID → COMPLETE or CANCELLED |     | NO  | 'PENDING'::character varying |
| subtotal_amount | integer |     |     | NO  |     |
| delivery_fee | integer |     |     | NO  | 0   |
| total_amount | integer | All amounts stored in smallest currency unit (IDR cents) |     | NO  |     |
| customer_name | varchar(255) |     |     | NO  |     |
| customer_phone | varchar(20) |     |     | NO  |     |
| delivery_type | varchar(20) |     |     | NO  |     |
| table_number | varchar(50) |     |     | YES |     |
| notes | text |     |     | YES |     |
| created_at | timestamp without time zone |     |     | NO  | now() |
| paid_at | timestamp without time zone |     |     | YES |     |
| completed_at | timestamp without time zone |     |     | YES |     |
| cancelled_at | timestamp without time zone |     |     | YES |     |
| session_id | varchar(255) |     |     | YES |     |
| ip_address | inet |     |     | YES |     |
| user_agent | text |     |     | YES |     |
| customer_email | varchar(255) |     |     | YES |     |

inventory_reservations

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| order_id | uuid |     |     | NO  |     |
| product_id | uuid |     |     | NO  |     |
| quantity | integer |     |     | NO  |     |
| status | varchar(20) | active: held, expired: TTL passed, converted: order paid, released: cancelled |     | NO  | 'active'::character varying |
| created_at | timestamp without time zone |     |     | NO  | now() |
| expires_at | timestamp without time zone | Background job marks expired reservations |     | NO  |     |
| released_at | timestamp without time zone |     |     | YES |     |

invitations

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| tenant_id | uuid |     |     | NO  |     |
| email | varchar(255) |     |     | NO  |     |
| role | varchar(20) |     |     | NO  |     |
| token | varchar(255) | Unique invitation token sent via email |     | NO  |     |
| status | varchar(20) |     |     | YES | 'pending'::character varying |
| invited_by | uuid | User who sent the invitation |     | NO  |     |
| expires_at | timestamp with time zone | Invitation expiration (typically 7 days) |     | NO  |     |
| accepted_at | timestamp with time zone |     |     | YES |     |
| created_at | timestamp with time zone |     |     | NO  | now() |
| updated_at | timestamp with time zone |     |     | NO  | now() |

notification_configs

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| tenant_id | uuid |     |     | NO  |     |
| order_notifications_enabled | boolean |     |     | NO  | true |
| test_mode | boolean | If true, all emails go to test_email only |     | NO  | false |
| test_email | varchar(255) |     |     | YES |     |
| created_at | timestamp with time zone |     |     | NO  | now() |
| updated_at | timestamp with time zone |     |     | NO  | now() |

notifications

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| tenant_id | uuid |     |     | NO  |     |
| user_id | uuid |     |     | YES |     |
| type | varchar(20) |     |     | NO  |     |
| status | varchar(20) |     |     | NO  | 'pending'::character varying |
| event_type | varchar(50) | Event that triggered notification (e.g., user.registered, user.login) |     | NO  |     |
| subject | varchar(255) |     |     | YES |     |
| body | text |     |     | NO  |     |
| recipient | varchar(255) |     |     | NO  |     |
| metadata | jsonb | Additional event data (JSON) |     | YES | '{}'::jsonb |
| sent_at | timestamp with time zone |     |     | YES |     |
| failed_at | timestamp with time zone |     |     | YES |     |
| error_msg | text |     |     | YES |     |
| retry_count | integer | Number of send attempts (max 3) |     | YES | 0   |
| max_retries | integer |     |     | YES | 3   |
| created_at | timestamp with time zone |     |     | NO  | now() |
| updated_at | timestamp with time zone |     |     | NO  | now() |

order_items

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| order_id | uuid |     |     | NO  |     |
| product_id | uuid |     |     | NO  |     |
| product_name | varchar(255) | Snapshot of product name at time of order (preserves history) |     | NO  |     |
| product_sku | varchar(100) |     |     | YES |     |
| quantity | integer |     |     | NO  |     |
| unit_price | integer |     |     | NO  |     |
| total_price | integer | Must equal quantity \* unit_price |     | NO  |     |
| created_at | timestamp without time zone |     |     | NO  | now() |

order_notes

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| order_id | uuid |     |     | NO  |     |
| note | text |     |     | NO  |     |
| created_by_user_id | uuid |     |     | YES |     |
| created_by_name | varchar(255) |     |     | YES |     |
| created_at | timestamp without time zone |     |     | NO  | now() |

order_settings

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| tenant_id | uuid |     |     | NO  |     |
| delivery_enabled | boolean |     |     | YES | true |
| pickup_enabled | boolean |     |     | YES | true |
| dine_in_enabled | boolean |     |     | YES | false |
| default_delivery_fee | integer |     |     | YES | 10000 |
| min_order_amount | integer |     |     | YES | 20000 |
| max_delivery_distance | numeric(10,2) |     |     | YES | 10.0 |
| estimated_prep_time | integer |     |     | YES | 30  |
| auto_accept_orders | boolean |     |     | YES | false |
| require_phone_verification | boolean |     |     | YES | false |
| created_at | timestamp with time zone |     |     | YES | now() |
| updated_at | timestamp with time zone |     |     | YES | now() |
| charge_delivery_fee | boolean | When false, the system will not charge delivery fees in orders. Useful when tenant uses external delivery services. |     | YES | true |

password_reset_tokens

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| tenant_id | uuid |     |     | NO  |     |
| user_id | uuid |     |     | NO  |     |
| token | varchar(255) | Unique reset token sent via email |     | NO  |     |
| expires_at | timestamp with time zone | Token expiration (typically 1 hour) |     | NO  |     |
| used_at | timestamp with time zone | When token was used (prevents reuse) |     | YES |     |
| created_at | timestamp with time zone |     |     | NO  | now() |

payment_transactions

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| order_id | uuid |     |     | NO  |     |
| midtrans_transaction_id | varchar(255) |     |     | YES |     |
| midtrans_order_id | varchar(255) |     |     | NO  |     |
| amount | integer |     |     | NO  |     |
| payment_type | varchar(50) |     |     | YES |     |
| transaction_status | varchar(50) |     |     | YES |     |
| fraud_status | varchar(50) |     |     | YES |     |
| notification_payload | jsonb | Full webhook payload for audit |     | YES |     |
| signature_key | varchar(512) |     |     | YES |     |
| signature_verified | boolean | Must be true before processing webhook |     | NO  | false |
| created_at | timestamp without time zone |     |     | NO  | now() |
| notification_received_at | timestamp without time zone |     |     | YES |     |
| settled_at | timestamp without time zone |     |     | YES |     |
| idempotency_key | varchar(255) | Prevents duplicate webhook processing: {midtrans_id}:{status} |     | YES |     |
| qr_code_url | text | URL to QRIS QR code image from Midtrans actions array |     | YES |     |
| qr_string | text | Raw QRIS string data for generating QR code |     | YES |     |
| expiry_time | timestamp without time zone | When the QRIS payment expires (default 15 minutes) |     | YES |     |

product_photos

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| product_id | uuid |     |     | NO  |     |
| tenant_id | uuid |     |     | NO  |     |
| storage_key | text | S3 object key following pattern: photos/{tenant_id}/{product_id}/{photo_id}\_{timestamp}.ext |     | NO  |     |
| original_filename | text |     |     | NO  |     |
| file_size_bytes | integer |     |     | NO  |     |
| mime_type | text |     |     | NO  |     |
| width_px | integer |     |     | YES |     |
| height_px | integer |     |     | YES |     |
| display_order | integer | Zero-based ordering for photo carousel display (0 = first photo) |     | NO  | 0   |
| is_primary | boolean | Primary photo displayed in product listings (only one per product) |     | NO  | false |
| created_at | timestamp without time zone |     |     | NO  | now() |
| updated_at | timestamp without time zone |     |     | NO  | now() |

products

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | uuid_generate_v4() |
| tenant_id | uuid |     |     | NO  |     |
| sku | varchar(50) |     |     | NO  |     |
| name | varchar(255) |     |     | NO  |     |
| description | text |     |     | YES |     |
| category_id | uuid |     |     | YES |     |
| selling_price | numeric(10,2) |     |     | NO  |     |
| cost_price | numeric(10,2) |     |     | NO  |     |
| tax_rate | numeric(5,2) |     |     | NO  | 0   |
| stock_quantity | integer |     |     | NO  | 0   |
| photo_path | varchar(500) |     |     | YES |     |
| photo_size | integer |     |     | YES |     |
| archived_at | timestamp without time zone |     |     | YES |     |
| created_at | timestamp without time zone |     |     | NO  | now() |
| updated_at | timestamp without time zone |     |     | NO  | now() |

schema_migrations

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| version | bigint |     | YES | NO  |     |
| dirty | boolean |     |     | NO  |     |

sessions

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| tenant_id | uuid |     |     | NO  |     |
| user_id | uuid |     |     | NO  |     |
| session_id | varchar(500) | JWT token or session identifier |     | NO  |     |
| expires_at | timestamp with time zone | Session expiration time (typically 24 hours) |     | NO  |     |
| ip_address | varchar(45) | IP address where session was created (security audit) |     | YES |     |
| user_agent | text |     |     | YES |     |
| created_at | timestamp with time zone |     |     | NO  | now() |
| terminated_at | timestamp with time zone |     |     | YES |     |

stock_adjustments

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | uuid_generate_v4() |
| tenant_id | uuid |     |     | NO  |     |
| product_id | uuid |     |     | NO  |     |
| user_id | uuid |     |     | NO  |     |
| previous_quantity | integer |     |     | NO  |     |
| new_quantity | integer |     |     | NO  |     |
| quantity_delta | integer |     |     | NO  |     |
| reason | varchar(50) |     |     | NO  |     |
| notes | text |     |     | YES |     |
| created_at | timestamp without time zone |     |     | NO  | now() |

tenant_configs

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| tenant_id | uuid |     |     | NO  |     |
| enabled_delivery_types | text\[\] | Array of: pickup, delivery, dine_in |     | NO  | '{pickup}'::text\[\] |
| service_area_type | varchar(20) |     |     | YES |     |
| service_area_data | jsonb | Flexible JSONB: radius={center:{lat,lng},radius_km:5} OR polygon={coordinates:\[\[lat,lng\],...\]} |     | YES |     |
| enable_delivery_fee_calculation | boolean | Tenant can disable automatic delivery fee calculation |     | YES | true |
| delivery_fee_type | varchar(20) |     |     | YES |     |
| delivery_fee_config | jsonb | Flexible JSONB pricing rules (see research.md) |     | YES |     |
| inventory_reservation_ttl_minutes | integer |     |     | YES | 15  |
| min_order_amount | integer |     |     | YES | 0   |
| location_lat | numeric(10,8) |     |     | YES |     |
| location_lng | numeric(11,8) |     |     | YES |     |
| created_at | timestamp without time zone |     |     | NO  | now() |
| updated_at | timestamp without time zone |     |     | NO  | now() |
| midtrans_server_key | text | Tenant-specific Midtrans server key (encrypted in production) |     | YES |     |
| midtrans_client_key | text | Tenant-specific Midtrans client key for frontend |     | YES |     |
| midtrans_merchant_id | text | Tenant-specific Midtrans merchant ID |     | YES |     |
| midtrans_environment | varchar(20) | Midtrans environment: sandbox or production |     | YES | 'sandbox'::character varying |

tenants

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| business_name | varchar(255) |     |     | NO  |     |
| slug | varchar(255) | URL-friendly unique identifier for tenant |     | NO  |     |
| status | varchar(20) | Tenant account status: active, suspended, or deleted |     | YES | 'inactive'::character varying |
| created_at | timestamp with time zone |     |     | NO  | now() |
| updated_at | timestamp with time zone |     |     | NO  | now() |
| storage_used_bytes | bigint | Total storage used by tenant in bytes (sum of all product photo file sizes) |     | NO  | 0   |
| storage_quota_bytes | bigint | Storage quota limit for tenant in bytes (default 5GB) |     | NO  | '5368709120'::bigint |

users

| **Column** | **Type** | **Comment** | **PK** | **Nullable** | **Default** |
| --- | --- | --- | --- | --- | --- |
| id  | uuid |     | YES | NO  | gen_random_uuid() |
| tenant_id | uuid | Foreign key to tenant - ensures data isolation |     | NO  |     |
| email | varchar(255) |     |     | NO  |     |
| password_hash | varchar(255) |     |     | NO  |     |
| first_name | varchar(50) |     |     | YES |     |
| last_name | varchar(50) |     |     | YES |     |
| locale | varchar(10) |     |     | NO  | 'en'::character varying |
| role | varchar(20) | User role: owner (full access), manager (limited), cashier (POS only) |     | NO  | 'cashier'::character varying |
| status | varchar(20) |     |     | YES | 'inactive'::character varying |
| email_verified | boolean | Whether user has verified their email address |     | YES | false |
| email_verified_at | timestamp with time zone |     |     | YES |     |
| verification_token | varchar(255) | Token for email verification (expires in 24h) |     | YES |     |
| verification_token_expires_at | timestamp with time zone |     |     | YES |     |
| last_login_at | timestamp with time zone |     |     | YES |     |
| created_at | timestamp with time zone |     |     | NO  | now() |
| updated_at | timestamp with time zone |     |     | NO  | now() |
| notification_email_enabled | boolean |     |     | NO  | true |
| notification_in_app_enabled | boolean |     |     | NO  | true |
| receive_order_notifications | boolean | Whether user receives email notifications for paid orders |     | YES | false |