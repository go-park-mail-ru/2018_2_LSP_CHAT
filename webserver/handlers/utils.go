package handlers

import (
	"io/ioutil"
	"mime/multipart"
	"os"
	"strconv"

	"github.com/go-park-mail-ru/2018_2_LSP_CHAT/user"
)

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func extractFields(u user.User, fieldsToReturn []string) map[string]interface{} {
	answer := map[string]interface{}{}
	for _, f := range fieldsToReturn {
		switch f {
		case "id":
			answer["id"] = u.ID
		case "firstname":
			answer["firstname"] = u.FirstName
		case "lastname":
			answer["lastname"] = u.LastName
		case "email":
			answer["email"] = u.Email
		case "username":
			answer["username"] = u.Username
		case "rating":
			answer["rating"] = u.Rating
		case "avatar":
			answer["avatar"] = u.Avatar
		}
	}
	return answer
}

func saveFile(file multipart.File, handle *multipart.FileHeader, id int) error {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(os.Getenv("AVATARS_PATH")+strconv.Itoa(id)+"_"+handle.Filename, data, 0666)
	if err != nil {
		return err
	}

	return nil
}
