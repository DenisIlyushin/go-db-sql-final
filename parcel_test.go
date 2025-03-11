package main

import (
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	randSource = rand.NewSource(time.Now().UnixNano())
	randRange  = rand.New(randSource)
)

func getTestParcel() Parcel {
	return Parcel{
		Client:    1000,
		Status:    ParcelStatusRegistered,
		Address:   "test",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", "tracker.db")
	require.NoError(t, err)
	db.SetMaxOpenConns(1) // Ограничиваем соединение до 1, чтобы избежать блокировки
	// Очищаем таблицу перед тестом
	_, err = db.Exec("DELETE FROM parcel")
	require.NoError(t, err)

	return db
}

func ConnectDb() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "tracker.db")
	return db, err
}

func TestAddGetDelete(t *testing.T) {
	// prepare
	db, err := ConnectDb()
	if err != nil {
		require.NoError(t, err)
	}
	defer db.Close()

	store := NewParcelStore(db)
	parcel := getTestParcel()

	parcel.Number, err = store.Add(parcel)

	require.NoError(t, err)
	require.Greater(t, parcel.Number, 0)

	stored, err := store.Get(parcel.Number)

	require.NoError(t, err)
	assert.Equal(t, parcel, stored)

	err = store.Delete(parcel.Number)
	require.NoError(t, err)

	_, err = store.Get(parcel.Number)
	require.ErrorIs(t, sql.ErrNoRows, err)
}

func TestSetAddress(t *testing.T) {
	db, err := ConnectDb()
	if err != nil {
		require.NoError(t, err)
	}
	defer db.Close()
	store := NewParcelStore(db)

	parcel := getTestParcel()
	num, err := store.Add(parcel)
	require.NoError(t, err)
	require.NotEmpty(t, num)

	newAddress := "new test address"
	err = store.SetAddress(num, newAddress)
	require.NoError(t, err)

	newParcel, err := store.Get(num)
	require.NoError(t, err)
	assert.Equal(t, newAddress, newParcel.Address)

	err = store.SetStatus(num, ParcelStatusSent)
	require.NoError(t, err)

	err = store.SetAddress(num, "another address")
	require.Error(t, err) // Должна быть ошибка
}

func TestSetStatus(t *testing.T) {
	db, err := ConnectDb()
	if err != nil {
		require.NoError(t, err)
	}
	defer db.Close()
	store := NewParcelStore(db)

	parcel := getTestParcel()
	num, err := store.Add(parcel)
	require.NoError(t, err)
	require.NotEmpty(t, num)

	newStatus := ParcelStatusSent
	err = store.SetStatus(num, newStatus)
	require.NoError(t, err)

	newParcel, err := store.Get(num)
	require.NoError(t, err)
	assert.Equal(t, newStatus, newParcel.Status)
}

func TestGetByClient(t *testing.T) {
	db, err := ConnectDb()
	if err != nil {
		require.NoError(t, err)
	}
	defer db.Close()
	store := NewParcelStore(db)

	parcels := []Parcel{
		getTestParcel(),
		getTestParcel(),
		getTestParcel(),
	}

	client := randRange.Intn(10_000_000)
	for i := range parcels {
		parcels[i].Client = client
	}

	for i, parcel := range parcels {
		id, err := store.Add(parcel)
		require.NoError(t, err)
		require.NotEmpty(t, id)
		parcels[i].Number = id
	}

	storedParcels, err := store.GetByClient(client)
	require.NoError(t, err)
	assert.ElementsMatch(t, storedParcels, parcels)
}
