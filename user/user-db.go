package user

import "database/sql"

type User struct {
	ID                     int
	CurrentActiveSessionID int
}

type UserService struct {
	DB *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{
		DB: db,
	}
}

func (us *UserService) GetUser(id int) (User, error) {
	user := User{}
	row := us.DB.QueryRow(`SELECT user_id, user_active_session_id FROM user WHERE user_id=$1`, id)

	err := row.Scan(&user.ID, &user.CurrentActiveSessionID)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (us *UserService) InsertNewUser(activeSessionID int) (User, error) {
	user := User{
		CurrentActiveSessionID: activeSessionID,
	}
	row := us.DB.QueryRow(
		`INSERT INTO user (user_active_session_id) VALUES ($1) RETURNING user_id`,
		activeSessionID,
	)
	err := row.Scan(&user.ID)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (us *UserService) UpdateUserCurrentActiveSession(
	userID int,
	activeSessionID int,
) (User, error) {
	_, err := us.DB.Exec(
		`UPDATE user SET user_active_session_id=$1 WHERE user_id=$2`,
		activeSessionID, userID,
	)
	if err != nil {
		return User{}, err
	}

	return User{CurrentActiveSessionID: activeSessionID, ID: userID}, nil
}
