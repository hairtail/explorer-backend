package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spacemeshos/go-spacemesh/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/spacemeshos/explorer-backend/model"
	"github.com/spacemeshos/explorer-backend/utils"
)

func (s *Storage) InitSmeshersStorage(ctx context.Context) error {
	_, err := s.db.Collection("smeshers").Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetName("idIndex").SetUnique(true)})
	if err != nil {
		return fmt.Errorf("error init `smeshers` collection: %w", err)
	}
	_, err = s.db.Collection("coinbases").Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "address", Value: 1}}, Options: options.Index().SetName("addressIndex").SetUnique(true)})
	if err != nil {
		return fmt.Errorf("error init `coinbases` collection: %w", err)
	}
	return nil
}

func (s *Storage) GetSmesher(parent context.Context, query *bson.D) (*model.Smesher, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	cursor, err := s.db.Collection("smeshers").Find(ctx, query)
	if err != nil {
		log.Info("GetSmesher: %v", err)
		return nil, err
	}
	if !cursor.Next(ctx) {
		log.Info("GetSmesher: Empty result")
		return nil, errors.New("Empty result")
	}
	doc := cursor.Current
	smesher := &model.Smesher{
		Id:             utils.GetAsString(doc.Lookup("id")),
		Name:           utils.GetAsString(doc.Lookup("name")),
		Lon:            doc.Lookup("lon").Double(),
		Lat:            doc.Lookup("lat").Double(),
		CommitmentSize: utils.GetAsUInt64(doc.Lookup("cSize")),
		Coinbase:       utils.GetAsString(doc.Lookup("coinbase")),
		AtxCount:       utils.GetAsUInt32(doc.Lookup("atxcount")),
		Timestamp:      utils.GetAsUInt32(doc.Lookup("timestamp")),
	}
	return smesher, nil
}

func (s *Storage) GetSmesherByCoinbase(parent context.Context, coinbase string) (*model.Smesher, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	cursor, err := s.db.Collection("coinbases").Find(ctx, &bson.D{{Key: "address", Value: coinbase}})
	if err != nil {
		log.Info("GetSmesherByCoinbase: %v", err)
		return nil, err
	}
	if !cursor.Next(ctx) {
		log.Info("GetSmesherByCoinbase: Empty result")
		return nil, errors.New("Empty result")
	}
	doc := cursor.Current
	smesher := utils.GetAsString(doc.Lookup("smesher"))
	if smesher == "" {
		log.Info("GetSmesherByCoinbase: Empty result")
		return nil, errors.New("Empty result")
	}
	return s.GetSmesher(ctx, &bson.D{{Key: "id", Value: smesher}})
}

func (s *Storage) GetSmeshersCount(parent context.Context, query *bson.D, opts ...*options.CountOptions) int64 {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	count, err := s.db.Collection("smeshers").CountDocuments(ctx, query, opts...)
	if err != nil {
		log.Info("GetSmeshersCount: %v", err)
		return 0
	}
	return count
}

func (s *Storage) IsSmesherExists(parent context.Context, smesher string) bool {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	count, err := s.db.Collection("smeshers").CountDocuments(ctx, bson.D{{Key: "id", Value: smesher}})
	if err != nil {
		log.Info("IsSmesherExists: %v", err)
		return false
	}
	return count > 0
}

func (s *Storage) GetSmeshers(parent context.Context, query *bson.D, opts ...*options.FindOptions) ([]bson.D, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	cursor, err := s.db.Collection("smeshers").Find(ctx, query, opts...)
	if err != nil {
		log.Info("GetSmeshers: %v", err)
		return nil, err
	}
	var docs interface{} = []bson.D{}
	err = cursor.All(ctx, &docs)
	if err != nil {
		log.Info("GetSmeshers: %v", err)
		return nil, err
	}
	if len(docs.([]bson.D)) == 0 {
		return nil, nil
	}
	return docs.([]bson.D), nil
}

func (s *Storage) SaveSmesher(parent context.Context, in *model.Smesher) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	_, err := s.db.Collection("smeshers").InsertOne(ctx, bson.D{
		{Key: "id", Value: in.Id},
		{Key: "name", Value: in.Name},
		{Key: "lon", Value: in.Lon},
		{Key: "lat", Value: in.Lat},
		{Key: "cSize", Value: in.CommitmentSize},
		{Key: "coinbase", Value: in.Coinbase},
		{Key: "atxcount", Value: in.AtxCount},
		{Key: "timestamp", Value: in.Timestamp},
	})
	if err != nil {
		return fmt.Errorf("error save smesher: %w", err)
	}
	return nil
}

func (s *Storage) UpdateSmesher(parent context.Context, smesher string, coinbase string, space uint64, timestamp uint32) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	_, err := s.db.Collection("coinbases").InsertOne(ctx, bson.D{
		{Key: "address", Value: coinbase},
		{Key: "smesher", Value: smesher},
	})
	if err != nil {
		return fmt.Errorf("error insert smesher into `coinbases`: %w", err)
	}
	atxCount, err := s.db.Collection("activations").CountDocuments(ctx, &bson.D{{Key: "smesher", Value: smesher}})
	if err != nil {
		log.Info("UpdateSmesher: GetActivationsCount: %v", err)
	}
	_, err = s.db.Collection("smeshers").UpdateOne(ctx, bson.D{{Key: "id", Value: smesher}}, bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "cSize", Value: space},
			{Key: "coinbase", Value: coinbase},
			{Key: "atxcount", Value: atxCount},
			{Key: "timestamp", Value: timestamp},
		}},
	})
	if err != nil {
		log.Info("UpdateSmesher: %v", err)
	}
	return err
}
