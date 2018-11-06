package user

import (
	"database/sql"
	"errors"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"golang.org/x/crypto/bcrypt"
)

// User Structure that stores user information retrieved from database or
// entered by user during registration
type User struct {
	ID          int    `json:"id"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	OldPassword string `json:"oldpassword"`
	Rating      int    `json:"rating"`
	Token       string `json:"token"`
	Username    string `json:"username"`
	FirstName   string `json:"firstname"`
	LastName    string `json:"lastname"`
	Avatar      string `json:"avatar"`
}

// GetAll return information about all user by pages (10 items per page)
func GetAll(db *sql.DB, page int, orderby string) ([]User, error) {
	if len(orderby) == 0 {
		orderby = "id"
	}
	rows, err := db.Query("SELECT id, username, email, firstname, lastname, rating, avatar FROM users ORDER BY "+orderby+" DESC LIMIT 10 OFFSET $1", page*10)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var firstname sql.NullString
	var lastname sql.NullString
	var avatar sql.NullString
	users := make([]User, 0)
	for rows.Next() {
		var u User
		err = rows.Scan(&u.ID, &u.Username, &u.Email, &firstname, &lastname, &u.Rating, &avatar)
		if err != nil {
			return nil, err
		}
		if temp, err := firstname.Value(); temp != nil && err == nil {
			u.FirstName = temp.(string)
		}
		if temp, err := lastname.Value(); temp != nil && err == nil {
			u.LastName = temp.(string)
		}
		if temp, err := avatar.Value(); temp != nil && err == nil {
			u.Avatar = temp.(string)
		}
		users = append(users, u)
	}
	return users, err
}

func (u *User) UpdateOne(db *sql.DB, data map[string]string) error {
	request := "UPDATE users SET "

	for k, v := range data {
		request += k + "='" + v + "',"
	}
	request = request[:len(request)-1]
	request += " WHERE id = $1 RETURNING id, username, email, firstname, lastname, rating, avatar"
	rows, err := db.Query(request, u.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return errors.New("User not found")
	}
	var firstname sql.NullString
	var lastname sql.NullString
	var avatar sql.NullString
	err = rows.Scan(&u.ID, &u.Username, &u.Email, &firstname, &lastname, &u.Rating, &avatar)
	if err != nil {
		return err
	}
	if temp, err := firstname.Value(); temp != nil && err == nil {
		u.FirstName = temp.(string)
	}
	if temp, err := lastname.Value(); temp != nil && err == nil {
		u.LastName = temp.(string)
	}
	if temp, err := avatar.Value(); temp != nil && err == nil {
		u.Avatar = temp.(string)
	}
	return nil
}

func GetOne(db *sql.DB, id int) (User, error) {
	var u User
	rows, err := db.Query("SELECT id, username, email, firstname, lastname, rating, avatar FROM users WHERE id = $1 LIMIT 1", id)
	if err != nil {
		return u, err
	}

	defer rows.Close()

	if !rows.Next() {
		return u, errors.New("User not found")
	}

	var firstname sql.NullString
	var lastname sql.NullString
	var avatar sql.NullString
	err = rows.Scan(&u.ID, &u.Username, &u.Email, &firstname, &lastname, &u.Rating, &avatar)
	if err != nil {
		return u, err
	}
	if temp, err := firstname.Value(); temp != nil && err == nil {
		u.FirstName = temp.(string)
	}
	if temp, err := lastname.Value(); temp != nil && err == nil {
		u.LastName = temp.(string)
	}
	if temp, err := avatar.Value(); temp != nil && err == nil {
		u.Avatar = temp.(string)
	}
	return u, err
}

// Register Function that sign ups user
func (u *User) Register(db *sql.DB) error {
	var err error
	// TODO чуть поправить валидацию
	if err := validateRegisterUnique(db, u); err != nil {
		return err
	}

	if u.Password, err = HashPassword(u.Password); err != nil {
		return nil
	}

	if err := u.createUser(db); err != nil {
		return err
	}

	if err := u.generateToken(); err != nil {
		return err
	}

	return nil
}

func ValidateUserPassword(db *sql.DB, password string, id int) (bool, error) {
	row, err := db.Query("SELECT password FROM users WHERE id = $1", id)
	if err != nil {
		return false, err
	}
	var hashedPassword string

	defer row.Close()
	if !row.Next() {
		return false, errors.New("User not found")
	}

	err = row.Scan(&hashedPassword)
	if err != nil {
		return false, err
	}
	return comparePasswords(hashedPassword, password), nil
}

func comparePasswords(hashedPwd string, plainPwd string) bool {
	byteHash := []byte(hashedPwd)
	bytePwd := []byte(plainPwd)
	err := bcrypt.CompareHashAndPassword(byteHash, bytePwd)
	if err != nil {
		return false
	}
	return true
}

func (u *User) generateToken() error {
	var err error
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":        u.ID,
		"generated": time.Now(),
	})
	u.Token, err = token.SignedString([]byte("abcderfs334f34r3we34"))
	return err
}

func validateRegisterUnique(db *sql.DB, u *User) error {
	rows, err := db.Query("SELECT EXISTS (SELECT * FROM users WHERE email = $1 LIMIT 1) AS email, EXISTS (SELECT * FROM users WHERE username = $2 LIMIT 1) AS username", u.Email, u.Username)
	if err != nil {
		return err
	}

	defer rows.Close()
	if !rows.Next() {
		return errors.New("User not found")
	}

	emailTaken, usernameTaken := false, false
	if err = rows.Scan(&emailTaken, &usernameTaken); err != nil {
		return err
	}

	if emailTaken {
		return errors.New("Email is already taken")
	}
	if usernameTaken {
		return errors.New("Username is already taken")
	}

	return nil
}

func (u *User) createUser(db *sql.DB) error {
	rows, err := db.Query("INSERT INTO users (firstname, lastname, email, password, username) VALUES ($1, $2, $3, $4, $5) RETURNING id;", u.FirstName, u.LastName, u.Email, u.Password, u.Username)
	if err != nil {
		return err
	}

	defer rows.Close()
	rows.Next()

	err = rows.Scan(&u.ID)
	return err
}

func HashPassword(pwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
