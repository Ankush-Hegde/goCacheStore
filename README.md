# goCacheStore

### Installation

# Run <code>go get github.com/Ankush-Hegde/goCacheStore</code>

### Usage

<code> Cache "github.com/Ankush-Hegde/goCacheStore" </code>

for database cache storage:
<code>
		_, cacheErr := Cache.MySQL(<<DbEndpoint>>, "<<tablename>>", "/", <<SessionLifetime>>, []byte("<SecretKey>"))

		if cacheErr != nil {
            return cacheErr
		}

		if cacheErr != nil {
			defer Cache.DatabaseCacheStore.Close()
		}
</code>

for filebased cache storage
<code>
		CacheFilePath := filepath.Join("path", "to_store_file")

		Cache.File(CacheFilePath, "/", Config.SessionLifetime, []byte("<SecretKey>"))
</code>

below is the code to create, get and forget the cache data,
key must be unique,
data must be the map[string]interface{},
time must be the int in sec,

<code>
Cache.CacheStorage.New(key, data, time)

Cache.CacheStorage.Get(key)

Cache.CacheStorage.Forget(key)
</code>