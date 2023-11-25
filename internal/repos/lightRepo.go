package repos

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/wheelibin/hugh/internal/constants"
	"github.com/wheelibin/hugh/internal/models"
)

const initSchema = `
  CREATE TABLE IF NOT EXISTS light (
    serviceid_light VARCHAR(36) PRIMARY KEY, 
    serviceid_zigbee VARCHAR(36), 
    name TEXT, 
    controlled_by_schedule VARCHAR(36),
    auto_on_from TEXT,
    auto_on_to TEXT,
    unreachable INTEGER,
    on_state INTEGER,
    target_brightness INTEGER,
    target_colour_temp INTEGER,
    target_on_state INTEGER,
    last_update_time TIMESTAMP,
    last_update_brightness INTEGER,
    last_update_colour_temp INTEGER,
    last_update_on_state INTEGER,
    override_brightness INTEGER,
    override_target_brightness INTEGER, -- target at time of override
    override_colour_temp INTEGER,
    override_target_colour_temp INTEGER, -- target at time of override
    override_time TIMESTAMP,
    override_on_state INTEGER,
    override_target_on_state INTEGER,    -- target at time of override
    min_colour_temp INTEGER,
    max_colour_temp INTEGER
  );

  CREATE TABLE IF NOT EXISTS scene (
    id VARCHAR(36) PRIMARY KEY,
    controlled_by_schedule VARCHAR(36),
    target_brightness INTEGER,
    target_colour_temp INTEGER,
    target_on_state INTEGER
  );

  DELETE FROM light;
  DELETE FROM scene;
`

type LightRepo struct {
	logger *log.Logger
	db     *sql.DB
}

func NewLightRepo(logger *log.Logger, db *sql.DB) (*LightRepo, error) {

	_, err := db.Exec(initSchema)
	if err != nil {
		return nil, fmt.Errorf("Error initialising light schema: %w", err)
	}

	return &LightRepo{logger: logger, db: db}, nil
}

func (r *LightRepo) Add(lights []models.HughLight) error {
	tx, _ := r.db.Begin()
	for _, light := range lights {
		_, err := tx.Exec(
			`INSERT INTO light 
      (serviceid_light, serviceid_zigbee, name, controlled_by_schedule, on_state, min_colour_temp, max_colour_temp, auto_on_from, auto_on_to) 
     VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);`,
			light.LightServiceId,
			light.ZigbeeServiceID,
			light.Name,
			light.ScheduleName,
			light.On,
			light.MinColorTemperatuerMirek,
			light.MaxColorTemperatuerMirek,
			light.AutoOnFrom,
			light.AutoOnTo,
		)
		if err != nil {
			return fmt.Errorf("Error adding light (%s): %w", light.Name, err)
		}
	}
	err := tx.Commit()
	if err != nil {
		return fmt.Errorf("Error adding lights: %w", err)
	}

	return nil
}

func (r *LightRepo) AddScenes(scenes []models.HughScene) error {
	tx, _ := r.db.Begin()
	for _, scene := range scenes {
		_, err := tx.Exec(
			`INSERT INTO scene 
      (id, controlled_by_schedule)
     VALUES ($1,$2);`,
			scene.ID,
			scene.ScheduleName,
		)
		if err != nil {
			return fmt.Errorf("Error adding scene (%s): %w", scene.ID, err)
		}
	}
	err := tx.Commit()
	if err != nil {
		return fmt.Errorf("Error adding scenes: %w", err)
	}

	return nil

}

func (r *LightRepo) SetLightOnState(lsID string, on bool) error {
	_, err := r.db.Exec("UPDATE light SET on_state = $1 WHERE serviceid_light = $2", on, lsID)
	if err != nil {
		return fmt.Errorf("Error setting light (%s) on state to %t: %w", lsID, on, err)
	}
	return nil
}

func (r *LightRepo) SetLightOnStateOverride(lsID string, on bool, targetOn bool) error {
	_, err := r.db.Exec("UPDATE light SET override_on_state = $1, on_state = $1, override_time = $2, override_target_on_state = $3 WHERE serviceid_light = $4", on, time.Now(), targetOn, lsID)
	if err != nil {
		return fmt.Errorf("Error setting light (%s) override on state to %t: %w", lsID, on, err)
	}
	return nil
}

func (r *LightRepo) SetLightBrightnessOverride(lsID string, brightness int, targetBrightness int) error {
	_, err := r.db.Exec("UPDATE light SET override_brightness = $1, override_time = $2, override_target_brightness = $3 WHERE serviceid_light = $4", brightness, time.Now(), targetBrightness, lsID)
	if err != nil {
		return fmt.Errorf("Error setting light (%s) override brightness to %v: %w", lsID, brightness, err)
	}
	return nil
}

func (r *LightRepo) SetLightColourTempOverride(lsID string, colourTemp int, targetColourTemp int) error {
	_, err := r.db.Exec("UPDATE light SET override_colour_temp = $1, override_time = $2, override_target_colour_temp = $3 WHERE serviceid_light = $4", colourTemp, time.Now(), targetColourTemp, lsID)
	if err != nil {
		return fmt.Errorf("Error setting light (%s) override colour temp to %v: %w", lsID, colourTemp, err)
	}
	return nil
}

func (r *LightRepo) SetLightUnreachable(lsID string) error {
	// when setting light to unreachable, also clear any overrides as they are now redundant
	_, err := r.db.Exec(`
    UPDATE light 
    SET unreachable = true,
      -- also clear manual override fields
        override_brightness = null, 
        override_colour_temp = null,
        override_target_brightness = null, 
        override_target_colour_temp = null, 
        override_target_on_state = null, 
        override_brightness = null, 
        override_time = null,
        override_on_state = null
    WHERE serviceid_light = $1`, lsID)
	if err != nil {
		return fmt.Errorf("Error setting light (%s) to unreachable: %w", lsID, err)
	}
	return nil
}

func (r *LightRepo) UpdateTargetState(scheduleName string, target models.LightState) error {
	_, err := r.db.Exec(
		`UPDATE light 
     SET target_brightness  = $1, 
         target_colour_temp = $2,
         target_on_state    = $3
     WHERE controlled_by_schedule = $4`,
		target.Brightness, target.TemperatureMirek, target.On, scheduleName)

	if err != nil {
		return fmt.Errorf("Error updating targets for lights in schedule (%s) to: %v: %w", scheduleName, target, err)
	}

	_, err = r.db.Exec(
		`UPDATE scene 
     SET target_brightness  = $1, 
         target_colour_temp = $2,
         target_on_state    = $3
     WHERE controlled_by_schedule = $4`,
		target.Brightness, target.TemperatureMirek, target.On, scheduleName)

	if err != nil {
		return fmt.Errorf("Error updating targets for scenes in schedule (%s) to: %v: %w", scheduleName, target, err)
	}

	return nil

}

func (r *LightRepo) GetLightTargetState(lsID string) (models.LightState, error) {
	row := r.db.QueryRow(`
    SELECT target_brightness, 
           target_colour_temp, 
           target_on_state, 
           min_colour_temp, 
           max_colour_temp,
           auto_on_from,
           auto_on_to,
           on_state
    FROM light 
    WHERE 
      serviceid_light = $1`, lsID)
	var (
		b        int
		t        int
		o        bool
		mint     int
		maxt     int
		autoOnFr string
		autoOnTo string
		on       bool
	)
	err := row.Scan(&b, &t, &o, &mint, &maxt, &autoOnFr, &autoOnTo, &on)
	if err != nil {
		return models.LightState{}, fmt.Errorf("Error reading target state for light (%s): %w", lsID, err)
	}

	// constrain the temperature values within the possible values for the particular light
	constrainedTemp := t
	if mint > 0 && t < mint {
		constrainedTemp = mint
	}
	if maxt > 0 && t > maxt {
		constrainedTemp = maxt
	}

	return models.LightState{
		Brightness:       b,
		TemperatureMirek: constrainedTemp,
		On:               o,
		AutoOnFrom:       autoOnFr,
		AutoOnTo:         autoOnTo,
		CurrentOnState:   on,
	}, nil
}

func (r *LightRepo) GetSceneTargetState(ID string) (models.LightState, error) {
	row := r.db.QueryRow("SELECT target_brightness, target_colour_temp, target_on_state FROM scene WHERE id = $1", ID)
	var (
		b int
		t int
		o bool
	)
	err := row.Scan(&b, &t, &o)
	if err != nil {
		return models.LightState{}, fmt.Errorf("Error reading target state for scene (%s): %w", ID, err)
	}
	return models.LightState{
		Brightness:       b,
		TemperatureMirek: t,
		On:               o,
	}, nil
}

func (r *LightRepo) IsScheduledLight(lsID string) (bool, error) {
	row := r.db.QueryRow("SELECT serviceid_light FROM light WHERE serviceid_light = $1", lsID)
	var id string
	err := row.Scan(&id)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		} else {
			return false, fmt.Errorf("Error reading target state for light (%s): %w", lsID, err)
		}
	}
	return true, nil
}

func (r *LightRepo) GetLightServiceIDForZigbeeID(zigbeeID string) (string, error) {
	row := r.db.QueryRow("SELECT serviceid_light FROM light WHERE serviceid_zigbee = $1", zigbeeID)
	var id string
	err := row.Scan(&id)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		} else {
			return "", fmt.Errorf("Error reading light service id for light with zigbeeID: %s: %w", zigbeeID, err)
		}
	}
	return id, nil
}

func (r *LightRepo) GetAllControllingLightIDs() ([]string, error) {

	rows, err := r.db.Query(`
    SELECT serviceid_light 
    FROM light 
    WHERE

      -- the target is different to current state
       (    target_brightness  != coalesce(last_update_brightness, -1)
         OR target_colour_temp != coalesce(last_update_colour_temp, -1) 
         OR target_on_state    != coalesce(last_update_on_state, -1)
       )
      -- and is reachable
      AND unreachable IS NULL

      AND (
        -- and the light doesn't have an override 
        (override_brightness IS NULL AND override_colour_temp IS NULL AND override_on_state IS NULL) 
        -- or the override was over [$1] minutes ago
        OR ((strftime('%s') - strftime('%s',override_time))/60 > $1)
      )
    `, constants.MaxLightOverrideMinutes)
	if err != nil {
		return nil, fmt.Errorf("Error reading ids for all lights: %w", err)
	}
	defer rows.Close()

	ids := []string{}

	for rows.Next() {
		var lsID string
		_ = rows.Scan(&lsID)

		ids = append(ids, lsID)
	}

	return ids, nil

}

func (r *LightRepo) GetAllSceneIDs() ([]string, error) {
	rows, err := r.db.Query("SELECT id FROM scene")
	if err != nil {
		return nil, fmt.Errorf("Error reading ids for all scenes: %w", err)
	}
	defer rows.Close()

	ids := []string{}

	for rows.Next() {
		var lsID string
		_ = rows.Scan(&lsID)

		ids = append(ids, lsID)
	}

	return ids, nil

}

func (r *LightRepo) MarkLightAsUpdated(lsID string) error {
	_, err := r.db.Exec(`
    UPDATE light 
    SET last_update_time = $1,
        last_update_brightness = target_brightness,
        last_update_colour_temp = target_colour_temp,
        last_update_on_state = target_on_state,
        unreachable = null, 
        -- also clear manual override fields
        override_brightness = null, 
        override_colour_temp = null,
        override_target_brightness = null, 
        override_target_colour_temp = null, 
        override_target_on_state = null, 
        override_brightness = null, 
        override_time = null,
        override_on_state = null
    WHERE serviceid_light = $2
  `, time.Now(), lsID)
	if err != nil {
		return fmt.Errorf("Error marking light (%s) as updated: %w", lsID, err)
	}
	return nil

}

func (r *LightRepo) GetLightLastUpdate(lsID string) (*time.Time, error) {
	row := r.db.QueryRow("SELECT last_update_time FROM light WHERE serviceid_light = $1", lsID)
	var lastUpdated time.Time
	err := row.Scan(&lastUpdated)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, fmt.Errorf("Error reading last update time for light (%s): %w", lsID, err)
		}
	}
	return &lastUpdated, nil
}
