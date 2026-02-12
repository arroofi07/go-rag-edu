# Troubleshooting Supabase Connection

## Problem Summary
Unable to connect to Supabase database from Go application. Tried multiple approaches:

### Attempted Solutions
1. ✗ Direct connection (`db.*.supabase.co:5432`) - DNS resolution fails ("no such host")
2. ✗ Session Pooler (`aws-0-ap-southeast-1.pooler.supabase.com:6543`) - Authentication fails ("Tenant or user not found" / "SQLSTATE XX000")
3. ✗ Switched from `lib/pq` to `pgx/v5` driver - Same errors persist
4. ✗ Added `pool_mode=session` parameter - No improvement

### Current Error
```
failed to ping database: dial tcp: lookup db.jzgmhmmzryrsdpsmmqar.supabase.co: no such host
```

## Next Steps to Resolve

### 1. Verify Supabase Project Status
- Go to: https://supabase.com/dashboard/project/jzgmhmmzryrsdpsmmqar
- Check if project is **ACTIVE** (not paused)
- If paused, click "Restore" or "Resume" button

### 2. Get Correct Connection String
In Supabase Dashboard:
1. Go to **Project Settings** → **Database**
2. Scroll to **Connection String** section
3. Select **Type: Golang**
4. Select **Method: Session pooler** (for IPv4 compatibility)
5. Click **"Pooler settings"** button to verify pooler mode
6. Copy the **complete connection string** shown

### 3. Verify Database Password
- In the connection string, replace `[YOUR-PASSWORD]` with your actual database password
- If you don't remember the password:
  1. Scroll down to **"Reset your database password"** section
  2. Click **"Database Settings"** link
  3. Reset the password
  4. Use the new password in your connection string

### 4. Update `.env` File
Once you have the correct connection string from Supabase dashboard:

```env
DATABASE_URL=<paste_the_complete_connection_string_here>
```

### 5. Test Connection
```bash
go run cmd/api/main.go
```

## Additional Debugging

### Test DNS Resolution
```bash
# Test if the hostname resolves
nslookup db.jzgmhmmzryrsdpsmmqar.supabase.co

# Test pooler hostname
nslookup aws-0-ap-southeast-1.pooler.supabase.com
```

### Test Network Connectivity
```bash
# Test if you can reach the database port
telnet db.jzgmhmmzryrsdpsmmqar.supabase.co 5432
# or
telnet aws-0-ap-southeast-1.pooler.supabase.com 6543
```

If DNS resolution or network connectivity fails, it might be a firewall or network configuration issue on your machine.
