package gorm_model

import "errors"

func primaryKeyNoBlankError() error {
	return errors.New("primary key is not blank")
}
