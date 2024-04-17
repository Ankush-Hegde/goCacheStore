package Cache

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
)

var DatabaseCacheStore *MySQLCacheStore

var FilebaseCacheStore *FileCacheStore

var CacheStorage Store

type Cache struct {
	SessionKey  string
	SessionData string
	IpAdress    string
	UserAgent   string
	CreatedAt   time.Time
	ModifiedAt  time.Time
	ExpiresAt   time.Time
}

type MySQLCacheStore struct {
	db         *sql.DB
	stmtInsert *sql.Stmt
	stmtDelete *sql.Stmt
	stmtUpdate *sql.Stmt
	stmtSelect *sql.Stmt

	table string
}

func MySQL(endpoint string, tableName string, path string, maxAge int, keyPairs ...[]byte) (*MySQLCacheStore, error) {
	db, err := sql.Open("mysql", endpoint)
	if err != nil {
		return nil, err
	}

	return MySQLCacheStoreFromConnection(db, tableName, path, maxAge, keyPairs...)
}

func MySQLCacheStoreFromConnection(db *sql.DB, tableName string, path string, maxAge int, keyPairs ...[]byte) (*MySQLCacheStore, error) {
	tableName = "`" + strings.Trim(tableName, "`") + "`"

	createTableQ := "CREATE TABLE IF NOT EXISTS " +
		tableName + " (session_key VARCHAR(40), " +
		"ip_adress VARCHAR(40), " +
		"user_agent VARCHAR(255), " +
		"session_data TEXT, " +
		"created_at TIMESTAMP DEFAULT NOW(), " +
		"modified_at TIMESTAMP NOT NULL DEFAULT NOW() ON UPDATE CURRENT_TIMESTAMP, " +
		"expires_at TIMESTAMP DEFAULT NOW(), PRIMARY KEY(`session_key`)) ENGINE=InnoDB;"

	if _, err := db.Exec(createTableQ); err != nil {
		switch err := err.(type) {
		case *mysql.MySQLError:
			// Error 1142 means permission denied for create command
			if err.Number == 1142 {
				break
			} else {
				return nil, err
			}
		default:
			return nil, err
		}
	}

	insQ := "INSERT INTO " + tableName +
		"(session_key, session_data, ip_adress, user_agent, created_at, modified_at, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?)"
	stmtInsert, stmtErr := db.Prepare(insQ)
	if stmtErr != nil {
		return nil, stmtErr
	}

	delQ := "DELETE FROM " + tableName + " WHERE session_key = ?"
	stmtDelete, stmtErr := db.Prepare(delQ)
	if stmtErr != nil {
		return nil, stmtErr
	}

	updQ := "UPDATE " + tableName + " SET session_data = ?, expires_at = ? " +
		"WHERE session_key = ?"
	stmtUpdate, stmtErr := db.Prepare(updQ)
	if stmtErr != nil {
		return nil, stmtErr
	}

	selQ := "SELECT session_key, ip_adress, user_agent, session_data, created_at, modified_at, expires_at from " +
		tableName + " WHERE session_key = ?"
	stmtSelect, stmtErr := db.Prepare(selQ)
	if stmtErr != nil {
		return nil, stmtErr
	}

	DatabaseCacheStore = &MySQLCacheStore{
		db:         db,
		stmtInsert: stmtInsert,
		stmtDelete: stmtDelete,
		stmtUpdate: stmtUpdate,
		stmtSelect: stmtSelect,

		table: tableName,
	}

	CacheStorage = DatabaseCacheStore

	return &MySQLCacheStore{
		db:         db,
		stmtInsert: stmtInsert,
		stmtDelete: stmtDelete,
		stmtUpdate: stmtUpdate,
		stmtSelect: stmtSelect,

		table: tableName,
	}, nil
}

func (m *MySQLCacheStore) Close() {
	m.stmtSelect.Close()
	m.stmtUpdate.Close()
	m.stmtDelete.Close()
	m.stmtInsert.Close()
	m.db.Close()
}

type Store interface {
	New(string, map[string]interface{}, int) (bool, error)

	Get(string) (map[string]interface{}, error)

	Forget(string) (bool, error)

	// Save(string, map[string]interface{}, int) (bool, error)

	// Flush() bool
}

func (m MySQLCacheStore) Get(session_key string) (map[string]interface{}, error) {
	row := m.stmtSelect.QueryRow(session_key)

	sess := Cache{}
	scanErr := row.Scan(&sess.SessionKey, &sess.IpAdress, &sess.UserAgent, &sess.SessionData, &sess.CreatedAt, &sess.ModifiedAt, &sess.ExpiresAt)
	//check if session exist if session didnt exist throw session expaired
	if scanErr != nil {
		return nil, errors.New("session expired")
	}
	if time.Until(sess.ExpiresAt) < 0 {
		return nil, errors.New("session expired")
	}

	//extending expire data for 1 week
	expires_at := time.Now().Add(time.Second * time.Duration(60*60*24*7))

	_, updErr := m.stmtUpdate.Exec(sess.SessionData, expires_at, sess.SessionKey)
	if updErr != nil {
		return nil, updErr
	}

	var fetchedSessionData map[string]interface{}

	err := json.Unmarshal([]byte(sess.SessionData), &fetchedSessionData)
	if err != nil {
		return nil, err
	}
	return fetchedSessionData, nil
}

func (m MySQLCacheStore) New(session_key string, data map[string]interface{}, sessionLifeTime int) (bool, error) {
	createdAt := time.Now()
	modifiedAt := time.Now()
	expiresAt := time.Now().Add(time.Second * time.Duration(sessionLifeTime))

	ip_adress := data["IP"]
	user_agent := data["userAgent"]

	data["expiresAt"] = expiresAt
	delete(data, "IP")
	delete(data, "userAgent")

	jsonData, err := json.Marshal(data)
	if err != nil {
		return false, err
	}

	//check for the duplicate entry for session key if duplicate key present change it
	_, insErr := m.stmtInsert.Exec(session_key, jsonData, ip_adress, user_agent, createdAt, modifiedAt, expiresAt)

	if insErr != nil {
		mysqlErr, ok := insErr.(*mysql.MySQLError)
		if ok && mysqlErr.Number == 1062 {
			return false, errors.New("please relogin")
		}
		return false, insErr
	}
	return true, nil
}

func (m MySQLCacheStore) Forget(session_key string) (bool, error) {
	_, delErr := m.stmtDelete.Exec(session_key)
	if delErr != nil {
		return false, delErr
	}
	return true, nil
}

// func (m MySQLCacheStore) Flush() bool {
// 	return true
// DELETE FROM `sessions` WHERE expires_at < NOW(); //this query is to delete all expaired session
// }

// func (m MySQLCacheStore) Save(id string, data map[string]interface{}, expair int) (bool, error) {

// 	return true, nil
// }

type FileCacheStore struct {
	filePath string
	mutex    sync.Mutex
}

func File(cacheFilePath string, path string, maxAge int, keyPairs ...[]byte) *FileCacheStore {
	FilebaseCacheStore = &FileCacheStore{
		filePath: cacheFilePath,
	}

	CacheStorage = FilebaseCacheStore

	return &FileCacheStore{
		filePath: cacheFilePath,
	}
}

func (s *FileCacheStore) New(key string, value map[string]interface{}, sessionLifeTime int) (bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	expiresAt := time.Now().Add(time.Second * time.Duration(sessionLifeTime))

	value["expiresAt"] = expiresAt

	jsonData, err := json.Marshal(value)
	if err != nil {
		return false, err
	}

	keyFile := filepath.Join(s.filePath, key)
	file, err := os.Create(keyFile)
	if err != nil {
		return false, err
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		return false, errors.New(err.Error())
	}

	return true, nil
}

func (s *FileCacheStore) Get(key string) (map[string]interface{}, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var result map[string]interface{}
	keyFile := filepath.Join(s.filePath, key)

	data, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *FileCacheStore) Forget(key string) (bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	keyFile := filepath.Join(s.filePath, key)

	err := os.Remove(keyFile)
	if err != nil {
		return false, err
	}

	return true, nil
}
