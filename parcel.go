package main

import (
	"database/sql"
	"fmt"
	"time"
)

type ParcelStore struct {
	db *sql.DB
}

func NewParcelStore(db *sql.DB) ParcelStore {
	return ParcelStore{db: db}
}

func (s ParcelStore) Add(p Parcel) (int, error) {
	stmt := `INSERT INTO parcel (client, status, address, created_at) VALUES (?, ?, ?, ?)`
	res, err := s.db.Exec(stmt, p.Client, p.Status, p.Address, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("ошибка при добавлении посылки: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("ошибка получения ID посылки: %w", err)
	}

	return int(id), nil
}

func (s ParcelStore) Get(number int) (Parcel, error) {
	var p Parcel
	stmt := `SELECT number, client, status, address, created_at FROM parcel WHERE number = ?`
	err := s.db.QueryRow(stmt, number).Scan(&p.Number, &p.Client, &p.Status, &p.Address, &p.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return p, fmt.Errorf("Посылка с номером %d не найдена", number)
			// для тестов было проще просто вернуть ошибку
			// return p, sql.ErrNoRows
		}
		return p, fmt.Errorf("ошибка при получении посылки: %w", err)
	}

	return p, nil
}

func (s ParcelStore) GetByClient(client int) ([]Parcel, error) {
	var parcels []Parcel
	stmt := `SELECT number, client, status, address, created_at FROM parcel WHERE client = ?`
	rows, err := s.db.Query(stmt, client)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении посылок клиента %d: %w", client, err)
	}
	defer rows.Close()

	for rows.Next() {
		var p Parcel
		if err := rows.Scan(&p.Number, &p.Client, &p.Status, &p.Address, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("ошибка при чтении данных: %w", err)
		}
		parcels = append(parcels, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при обработке строк: %w", err)
	}

	return parcels, nil
}

func (s ParcelStore) SetStatus(number int, status string) error {
	stmt := `UPDATE parcel SET status = ? WHERE number = ?`
	_, err := s.db.Exec(stmt, status, number)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении статуса посылки № %d: %w", number, err)
	}
	return nil
}

func (s ParcelStore) SetAddress(number int, address string) error {
	parcel, err := s.Get(number)
	if err != nil {
		return err
	}

	if parcel.Status != ParcelStatusRegistered {
		return fmt.Errorf("нельзя изменить адрес для посылки № %d, так как её статус — %s", number, parcel.Status)
	}

	stmt := `UPDATE parcel SET address = ? WHERE number = ?`
	_, err = s.db.Exec(stmt, address, number)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении адреса посылки № %d: %w", number, err)
	}
	return nil
}

func (s ParcelStore) Delete(number int) error {
	parcel, err := s.Get(number)
	if err != nil {
		return err
	}

	if parcel.Status != ParcelStatusRegistered {
		return fmt.Errorf("нельзя удалить посылку № %d, так как её статус — %s", number, parcel.Status)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка при начале транзакции: %w", err)
	}

	stmt := `DELETE FROM parcel WHERE number = ?`
	_, err = tx.Exec(stmt, number)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("ошибка при удалении посылки № %d: %w", number, err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("ошибка при фиксации удаления посылки № %d: %w", number, err)
	}

	return nil
}
