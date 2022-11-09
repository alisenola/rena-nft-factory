package Collection

import (
	"context"
	"fmt"

	"rena-nft-factory/database"
	"rena-nft-factory/model"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type CollectionDB interface {
	RunInTx(ctx context.Context, f func(ctx context.Context) error) error

	// SaveCollection saves a given collection.
	SaveCollection(collection *model.Collection) error

	// FindCollectionBySlug returns a collection with given slug
	// database.ErrNotFound error is returned if not exist
	FindCollectionBySlug(slug string) *model.Collection
}

type collectionDB struct {
	db *gorm.DB
}

func (a *collectionDB) RunInTx(ctx context.Context, f func(ctx context.Context) error) error {
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

func (a *collectionDB) SaveCollection(collection *model.Collection) error {
	db := a.db

	if err := db.Create(collection).Error; err != nil {
		if database.IsKeyConflictErr(err) {
			return database.ErrKeyConflict
		}
		return err
	}
	return nil
}

func (a *collectionDB) FindCollectionBySlug(slug string) *model.Collection {
	db := a.db

	var ret model.Collection
	err := db.First(&ret, "slug = ?", slug).Error

	if err != nil {
		if database.IsRecordNotFoundErr(err) {
			return nil
		}
		return nil
	}
	return &ret
}

// NewCollectionDB creates a new collection db with given db
func NewCollectionDB(db *gorm.DB) CollectionDB {
	return &collectionDB{
		db: db,
	}
}
