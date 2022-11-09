package Item

import (
	"context"
	"fmt"

	"rena-nft-factory/database"
	"rena-nft-factory/model"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type ItemDB interface {
	RunInTx(ctx context.Context, f func(ctx context.Context) error) error

	// SaveItem saves a given item.
	SaveItem(item *model.Item) error

	// FindItemByAddress returns a item with given address
	// database.ErrNotFound error is returned if not exist
	FindItemByAddress(address string, id int) *model.Item
}

type itemDB struct {
	db *gorm.DB
}

func (a *itemDB) RunInTx(ctx context.Context, f func(ctx context.Context) error) error {
	tx := a.db.Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "start tx")
	}

	ctx = database.WithDB(ctx, tx)
	if err := f(ctx); err != nil {
		if err1 := tx.Rollback().Error; err1 != nil {
			return errors.Wrap(err, fmt.Sprintf("rollback tx: %v", err1.Error()))
		}
		return errors.Wrap(err, "invoke function")
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("commit tx: %v", err)
	}
	return nil
}

func (a *itemDB) SaveItem(item *model.Item) error {
	db := a.db

	if err := db.Create(item).Error; err != nil {
		if database.IsKeyConflictErr(err) {
			return database.ErrKeyConflict
		}
		return err
	}
	return nil
}

func (a *itemDB) FindItemByAddress(address string, id int) *model.Item {
	db := a.db

	var ret model.Item
	err := db.First(&ret, "contract_address = ? AND token_id = ?", address, id).Error

	if err != nil {
		if database.IsRecordNotFoundErr(err) {
			return nil
		}
		return nil
	}
	return &ret
}

// NewItemDB creates a new item db with given db
func NewItemDB(db *gorm.DB) ItemDB {
	return &itemDB{
		db: db,
	}
}
