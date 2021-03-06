package storelib

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Configuration struct {
	Db      *sql.DB
	Timeout time.Duration
	Table   string
}

const defaultTimeout = 3 * time.Second

func New(configuration Configuration) *MariaDbStore {
	return &MariaDbStore{
		db:      configuration.Db,
		table:   configuration.Table,
		timeout: timeoutOrDefaultTimeout(configuration.Timeout),
	}
}

func timeoutOrDefaultTimeout(to time.Duration) time.Duration {
	if to == 0 {
		return defaultTimeout
	} else {
		return to
	}
}

type MariaDbStore struct {
	table   string
	db      *sql.DB
	timeout time.Duration
}

func (m *MariaDbStore) Init() error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), m.timeout)
	defer cancelFunc()
	_, err := m.db.ExecContext(ctx, fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (`key` varchar(255), `value` MEDIUMBLOB, CONSTRAINT PK PRIMARY KEY (`key`)) ;", m.table))
	return err
}

func (m *MariaDbStore) Set(key string, value []byte) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), m.timeout)
	defer cancelFunc()
	_, err := m.db.ExecContext(ctx, fmt.Sprintf("INSERT INTO `%s` (`key`, `value`) VALUES (?,?) ON DUPLICATE KEY UPDATE `value`=?", m.table), key, value, value)
	return err
}

func (m *MariaDbStore) Get(key string) ([]byte, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), m.timeout)
	defer cancelFunc()

	row := m.db.QueryRowContext(ctx, fmt.Sprintf("SELECT `value` FROM `%s` WHERE `key` = ?;", m.table), key)
	if err := row.Err(); err != nil {
		return nil, err
	}

	var result []byte
	if err := row.Scan(&result); err != nil {
		return nil, err
	} else {
		return result, nil
	}
}

func (m *MariaDbStore) Delete(key string) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), m.timeout)
	defer cancelFunc()

	result, err := m.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM `%s` WHERE `key` = ?", m.table), key)
	if err != nil {
		return err
	}

	if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows == 0 {
		return errors.New("unknown key")
	} else {
		return nil
	}
}

func (m *MariaDbStore) KeysWithSuffix(suffix string) ([]string, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), m.timeout)
	defer cancelFunc()

	rows, err := m.db.QueryContext(ctx, fmt.Sprintf("SELECT `key` FROM `%s` WHERE `key` LIKE ?;", m.table), "%"+suffix)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []string
	for cont := true; cont; cont = rows.NextResultSet() {
		for rows.Next() {
			var key string
			if err = rows.Scan(&key); err != nil {
				return nil, err
			}
			result = append(result, key)
		}
	}

	return result, nil
}
