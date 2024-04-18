# Dude!!
Testing out stuff man!


# Config
```.dotenv
# .env
DB_DEFAULT_URL=postgres://postgres:changeme@localhost/postgres?sslmode=disable&_x-poolSize=10
```

## table layout
Tables created on startup for detail on layout refer to [1_initial.up.sql](persistence/migrations/1_initial.up.sql)


# Usage
```bash
 curl -v -H 'X-DB-Name:DEFAULT' localhost:8080/dude
```




