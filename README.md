# goCacheStore

## Installation

Run <code>go get github.com/Ankush-Hegde/goCacheStore</code> from command line.

## Usage

```
import (
	Cache "github.com/Ankush-Hegde/goCacheStore"
)
```

### for database cache storage

```
DbEndpoint := UN:PASS@tcp(<IP>:<PORT>)/<DB>?parseTime=true&loc=Local
SessionLifetimeInSec := 1598453

_, cacheErr := Cache.MySQL(DbEndpoint, "tablename", "/", SessionLifetimeInSec, []byte("<SecretKey>"))

		if cacheErr != nil {
            		return cacheErr
		}
		if cacheErr != nil {
			defer Cache.DatabaseCacheStore.Close()
		}
```

### for filebased cache storage
```
CacheFilePath := filepath.Join("path", "to_store_file")
SessionLifetimeInSec := 1598453

Cache.File(CacheFilePath, "/", SessionLifetimeInSec, []byte("<SecretKey>"))
```
### below is the code to create, get and forget the cache data,<br>
```
Cache.CacheStorage.New(key, data, time)

data := Cache.CacheStorage.Get(key)

Cache.CacheStorage.Forget(key)
```
### note:-
key must be unique string,<br>
data must be the map[string]interface{},<br>
time must be the int in sec,<br>
