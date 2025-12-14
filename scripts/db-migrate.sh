# DUMP PostgreSQL Database to a File
pg_dump \
  -h localhost \
  -U pos_user \
  -Fc \
  --no-owner \
  --no-acl \
  -f pos_db.dump \
  pos_db \
  && sleep 10 && \ 
# RESTORE PostgreSQL Database from a File
pg_restore \
  -h aws-1-ap-south-1.pooler.supabase.com \
  -U postgres.uzixzhhkupleocwrswuy \
  -d postgres \
  --clean \
  --if-exists \
  --no-owner \
  --no-acl \
  pos_db.dump