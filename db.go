package main

import (
	_ "modernc.org/sqlite"
)

func (b *Bot) SaveMinimalSleepLog(record DbSleepRecord) error {

	// insert the record into the database
	_, err := b.Db.Exec(`
		INSERT OR REPLACE INTO sleep_logs_sent (data_point_name, date, efficiency, duration)
		VALUES (?, ?, ?, ?);
	`, record.DataPointName, record.Date, record.Efficiency, record.Duration)
	if err != nil {
		return err
	}

	return nil
}

func (b *Bot) HasRecordAlreadyBeenSent(recordName string) (bool, error) {

	res, err := b.Db.Query(`
		SELECT COUNT(*) FROM sleep_logs_sent WHERE data_point_name = ?;
	`, recordName)
	if err != nil {
		return false, err
	}
	defer res.Close()

	var count int
	if res.Next() {
		err = res.Scan(&count)
		if err != nil {
			return false, err
		}
	}

	return count > 0, nil
}

func (b *Bot) GetRecordsAfter(date string) ([]DbSleepRecord, error) {
	var records []DbSleepRecord
	res, err := b.Db.Query(`
		SELECT data_point_name, date, efficiency, duration FROM sleep_logs_sent WHERE date > ?;
	`, date)
	if err != nil {
		return records, err
	}
	defer res.Close()

	for res.Next() {
		var record DbSleepRecord
		err = res.Scan(&record.DataPointName, &record.Date, &record.Efficiency, &record.Duration)
		if err != nil {
			return records, err
		}
		records = append(records, record)
	}

	return records, nil
}
