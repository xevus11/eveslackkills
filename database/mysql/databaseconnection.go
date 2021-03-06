package mysql

import (
	"fmt"

	"github.com/morpheusxaut/eveslackkills/misc"
	"github.com/morpheusxaut/eveslackkills/models"

	// Blank import of the MySQL driver to use with sqlx
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// DatabaseConnection provides an implementation of the Connection interface using a MySQL database
type DatabaseConnection struct {
	// Config stores the current configuration values being used
	Config *misc.Configuration

	conn *sqlx.DB
}

// Connect tries to establish a connection to the MySQL backend, returning an error if the attempt failed
func (c *DatabaseConnection) Connect() error {
	conn, err := sqlx.Connect("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=true", c.Config.DatabaseUser, c.Config.DatabasePassword, c.Config.DatabaseHost, c.Config.DatabaseSchema))
	if err != nil {
		return err
	}

	c.conn = conn

	return nil
}

// RawQuery performs a raw MySQL query and returns a map of interfaces containing the retrieve data. An error is returned if the query failed
func (c *DatabaseConnection) RawQuery(query string, v ...interface{}) ([]map[string]interface{}, error) {
	rows, err := c.conn.Query(query, v...)
	if err != nil {
		return nil, err
	}

	columns, _ := rows.Columns()
	count := len(columns)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)

	var results []map[string]interface{}

	for rows.Next() {
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		rows.Scan(valuePtrs...)

		resultRow := make(map[string]interface{})

		for i, col := range columns {
			resultRow[col] = values[i]
		}

		results = append(results, resultRow)
	}

	return results, nil
}

// LoadAllCorporations retrieves all corporations from the MySQL database, returning an error if the query failed
func (c *DatabaseConnection) LoadAllCorporations() ([]*models.Corporation, error) {
	var corporations []*models.Corporation

	err := c.conn.Select(&corporations, "SELECT id, evecorporationid, lastkillid, lastlossid, name, killcomment, losscomment FROM corporations")
	if err != nil {
		return nil, err
	}

	for _, corporation := range corporations {
		ignoredRegions, err := c.LoadAllIgnoredRegionsForCorporation(corporation.ID)
		if err != nil {
			return nil, err
		}

		corporation.IgnoredRegions = ignoredRegions
	}

	return corporations, nil
}

// LoadCorporation retrieves the corporation with the given ID from the MySQL database, returning an error if the query failed
func (c *DatabaseConnection) LoadCorporation(corporationID int64) (*models.Corporation, error) {
	corporation := &models.Corporation{}

	err := c.conn.Get(corporation, "SELECT id, evecorporationid, lastkillid, lastlossid, name, killcomment, losscomment FROM corporations WHERE id=?", corporationID)
	if err != nil {
		return nil, err
	}

	ignoredRegions, err := c.LoadAllIgnoredRegionsForCorporation(corporation.ID)
	if err != nil {
		return nil, err
	}

	corporation.IgnoredRegions = ignoredRegions

	return corporation, nil
}

// LoadAllIgnoredRegionsForCorporation retrieves all ignored regions associated with the given corporation from the MySQL database, returning an error if the query failed
func (c *DatabaseConnection) LoadAllIgnoredRegionsForCorporation(corporationID int64) ([]int64, error) {
	var ignoredRegions []int64

	err := c.conn.Select(&ignoredRegions, "SELECT regionid FROM ignoredregions WHERE corporationID=?", corporationID)
	if err != nil {
		return nil, err
	}

	return ignoredRegions, nil
}

// QueryShipName looks up the given ship type ID in the MySQL database and returns the ship's name, returning an error if the query failed
func (c *DatabaseConnection) QueryShipName(shipTypeID int64) (string, error) {
	row := c.conn.QueryRowx("SELECT typeName FROM invTypes WHERE typeID=?", shipTypeID)

	var shipName string
	err := row.Scan(&shipName)
	if err != nil {
		return "", err
	}

	return shipName, nil
}

// QueryShipName looks up the given solar system ID in the MySQL database and returns the associated region ID, returning an error if the query failed
func (c *DatabaseConnection) QueryRegionID(solarSystemID int64) (int64, error) {
	row := c.conn.QueryRowx("SELECT regionID FROM mapSolarSystems WHERE solarSystemID=?", solarSystemID)

	var regionID int64
	err := row.Scan(&regionID)
	if err != nil {
		return -1, err
	}

	return regionID, nil
}

// SaveCorporation saves a corporation to the database, returning the updated model or an error if the query failed
func (c *DatabaseConnection) SaveCorporation(corporation *models.Corporation) (*models.Corporation, error) {
	if corporation.ID > 0 {
		_, err := c.conn.Exec("UPDATE corporations SET evecorporationid=?, lastkillid=?, lastlossid=? WHERE id=?", corporation.EVECorporationID, corporation.LastKillID, corporation.LastLossID, corporation.ID)
		if err != nil {
			return nil, err
		}
	} else {
		resp, err := c.conn.Exec("INSERT INTO corporations(evecorporationid, lastkillid, lastlossid) VALUES(?, ?, ?)", corporation.EVECorporationID, corporation.LastKillID, corporation.LastLossID)
		if err != nil {
			return nil, err
		}

		lastInsertedID, err := resp.LastInsertId()
		if err != nil {
			return nil, err
		}

		corporation.ID = lastInsertedID
	}

	return corporation, nil
}
