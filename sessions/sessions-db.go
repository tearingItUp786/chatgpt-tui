package sessions

import (
	"database/sql"
	"encoding/json"
	"log"
)

type Session struct {
	ID          int
	Messages    []MessageToSend
	CreatedAt   string
	SessionName string
}

type SessionService struct {
	DB *sql.DB
}

func NewSessionService(db *sql.DB) *SessionService {
	return &SessionService{
		DB: db,
	}
}

func (ss *SessionService) GetLatestSession() (Session, error) {
	var messages string
	session := Session{}
	row := ss.DB.QueryRow(`
SELECT id, messages, created_at, session_name FROM sessions ORDER BY created_at DESC LIMIT 1;
    `)
	err := row.Scan(&session.ID, &messages, &session.CreatedAt, &session.SessionName)
	if err != nil {
		if err == sql.ErrNoRows {
			// No session found, create a new one
			newSession := Session{
				// Initialize your session fields as needed
				// ID will be set by the database if using auto-increment
				SessionName: "New Session",     // Set a default or generate a name
				Messages:    []MessageToSend{}, // Assuming Messages is a slice of Message
			}
			// Insert the new session into the database
			// Insert the new session into the database
			insertSQL := `INSERT INTO sessions (session_name, messages) VALUES (?, ?);`
			messagesJSON, err := json.Marshal(newSession.Messages)
			if err != nil {
				return Session{}, err
			}
			result, err := ss.DB.Exec(
				insertSQL,
				newSession.SessionName,
				messagesJSON,
			)
			if err != nil {
				return Session{}, err
			}
			// Get the last inserted ID
			lastInsertID, err := result.LastInsertId()
			if err != nil {
				return Session{}, err
			}
			// Set the ID of the new session
			newSession.ID = int(lastInsertID)
			// Return the new session
			return newSession, nil
		} else {
			// An error occurred that isn't due to no rows being found
			return Session{}, err
		}
	}
	// If we reach this point, a session was found, so unmarshal the messages
	err = json.Unmarshal([]byte(messages), &session.Messages)
	if err != nil {
		return Session{}, err
	}
	// Return the found session
	return session, nil
}

// get me all the sessions
func (ss *SessionService) GetAllSessions() ([]Session, error) {
	rows, err := ss.DB.Query(`SELECT id,  created_at, session_name FROM sessions ORDER BY id DESC`)
	if err != nil {
		panic(err)
	}
	sessions := []Session{}
	for rows.Next() {
		aSession := Session{}
		rows.Scan(&aSession.ID, &aSession.CreatedAt, &aSession.SessionName)
		sessions = append(sessions, aSession)
	}

	return sessions, nil
}

func (ss *SessionService) UpdateSessionMessages(id int, messages []MessageToSend) {
	jsonData, err := json.Marshal(messages)
	if err != nil {
		// TODO: better error handling
		panic(err)
	}

	_, err = ss.DB.Exec(`
			UPDATE sessions
			SET messages  = $1
			where id = $2
	`, jsonData, id)

	if err != nil {
		// TODO: handle better
		panic(err)
	}
}

func (ss *SessionService) UpdateSessionName(id int, name string) {
	_, err := ss.DB.Exec(`
			UPDATE sessions
			SET session_name = $1
			where id = $2
	`, name, id)
	if err != nil {
		// TODO: handle better
		panic(err)
	}
}

func (ss *SessionService) InsertNewSession(name string, jsonData []byte) int {
	var sessionID int
	row := ss.DB.QueryRow(`
						INSERT INTO sessions (session_name, messages)
						VALUES ($1, $2) RETURNING id;`, name, jsonData)
	err := row.Scan(&sessionID)
	log.Println("session id", sessionID)
	if err != nil {
		// TODO: better error handling
		log.Println("error", err)
		panic(err)
	}

	return sessionID
}
