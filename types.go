package main

type Config struct {
	GoogleClientID           string
	GoogleClientSecret       string
	GoogleHealthRefreshToken string
}

type DbSleepRecord struct {
	DataPointName string `db:"data_point_name"` // primary key
	Date          string `db:"date"`            // "2026-07-09"
	Efficiency    int    `db:"efficiency"`
	Duration      int    `db:"duration"`
}
