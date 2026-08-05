package db

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	AccountID uint `gorm:"index" json:"account_id"`
}

var _ IDModel = &User{}

func (u *User) Save(ctx context.Context) error {
	return DB().WithContext(ctx).Save(u).Error
}

func (u *User) GetID() uint { return u.ID }

func (u *User) LoadRelation(_ context.Context, _ ...string) {
}

func GetUser(ctx context.Context, uid uint) *User {
	u := &User{}
	DB().WithContext(ctx).Where("id", uid).First(u)
	return u
}

func RegisterUser(ctx context.Context, accountID uint) (*User, error) {
	u := &User{}
	err := DB().WithContext(ctx).Where("account_id = ?", accountID).First(u).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if u.ID > 0 {
		return u, nil
	}
	u.AccountID = accountID
	err = u.Save(ctx)
	if err != nil {
		return nil, err
	}
	return u, nil
}
