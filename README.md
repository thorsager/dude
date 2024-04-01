# Dude!!
Testing out stuff man!


# Config
```.dotenv
# .env
DB_DEFAULT_URL=postgres://postgres:changeme@localhost/postgres?sslmode=disable&_x-poolSize=10
```

## table layout
```sql
CREATE TABLE IF NOT EXISTS dude (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
)
```                         

# Usage
```bash
 curl -v -H 'X-DB-Name:DEFAULT' localhost:8080/dude
```




