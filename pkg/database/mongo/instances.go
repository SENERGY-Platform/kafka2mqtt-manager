/*
 * Copyright 2020 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mongo

import (
	"context"
	"errors"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"log"
)

const idFieldName = "Id"
const nameFieldName = "Name"
const userIdFieldName = "UserId"
const createdAtFieldName = "CreatedAt"
const updatedAtFieldName = "UpdatedAt"
const generatedFieldName = "Generated"

var idKey string
var nameKey string
var ownerKey string
var createdAtKey string
var updatedAtKey string
var generatedKey string

func init() {
	var err error
	idKey, err = getBsonFieldName(model.Instance{}, idFieldName)
	if err != nil {
		log.Fatal(err)
	}
	nameKey, err = getBsonFieldName(model.Instance{}, nameFieldName)
	if err != nil {
		log.Fatal(err)
	}
	ownerKey, err = getBsonFieldName(model.Instance{}, userIdFieldName)
	if err != nil {
		log.Fatal(err)
	}
	createdAtKey, err = getBsonFieldName(model.Instance{}, createdAtFieldName)
	if err != nil {
		log.Fatal(err)
	}
	updatedAtKey, err = getBsonFieldName(model.Instance{}, updatedAtFieldName)
	if err != nil {
		log.Fatal(err)
	}
	generatedKey, err = getBsonFieldName(model.Instance{}, generatedFieldName)
	if err != nil {
		log.Fatal(err)
	}

	CreateCollections = append(CreateCollections, func(db *Mongo) error {
		collection := db.client.Database(db.config.MongoTable).Collection(db.config.MongoImportTypeCollection)
		err = db.ensureCompoundIndex(collection, "instanceOwnerIdindex", true, true, ownerKey, idKey)
		if err != nil {
			return err
		}
		return nil
	})
}

func (this *Mongo) instanceCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoImportTypeCollection)
}

func (this *Mongo) GetInstance(ctx context.Context, id string, owner string) (instance model.Instance, exists bool, err error) {
	result := this.instanceCollection().FindOne(ctx, bson.M{ownerKey: owner, idKey: id})
	err = result.Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return instance, false, errors.New("requested instance nonexistent")
		}
		return instance, false, err
	}
	err = result.Decode(&instance)
	if err == mongo.ErrNoDocuments {
		return instance, false, nil
	}
	return instance, true, err
}

func (this *Mongo) ListInstances(ctx context.Context, limit int64, offset int64, sort string, owner string, asc bool, search string, includeGenerated bool) (result []model.Instance, total int64, err error) {
	opt := options.Find()
	opt.SetLimit(limit)
	opt.SetSkip(offset)

	sortby := idKey
	switch sort {
	case "id":
		sortby = idKey
	case "name":
		sortby = nameKey
	case "created_at":
		sortby = createdAtKey
	case "updated_at":
		sortby = updatedAtKey
	default:
		sortby = idKey
	}
	direction := int32(1)
	if !asc {
		direction = int32(-1)
	}
	opt.SetSort(bsonx.Doc{{sortby, bsonx.Int32(direction)}})
	var filter bson.M
	if includeGenerated {
		filter = bson.M{ownerKey: owner, nameKey: primitive.Regex{
			Pattern: ".*" + search + ".*",
		}}
	} else {
		// filter for generatedKey == False || generatedKey == undefined to find legacy instances
		filter = bson.M{ownerKey: owner, "$or": []bson.M{{generatedKey: false}, {generatedKey: bson.M{"$exists": false}}},
			nameKey: primitive.Regex{
				Pattern: ".*" + search + ".*",
			}}
	}
	cursor, err := this.instanceCollection().Find(ctx, filter, opt)
	if err != nil {
		return nil, 0, err
	}
	for cursor.Next(context.Background()) {
		instance := model.Instance{}
		err = cursor.Decode(&instance)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, instance)
	}
	if cursor.Err() != nil {
		return nil, 0, cursor.Err()
	}
	total, err = this.instanceCollection().CountDocuments(context.Background(), bson.M{ownerKey: owner})
	return
}

func (this *Mongo) SetInstance(ctx context.Context, instance model.Instance, owner string) error {
	_, err := this.instanceCollection().ReplaceOne(ctx, bson.M{ownerKey: owner, idKey: instance.Id}, instance, options.Replace().SetUpsert(true))
	return err
}

func (this *Mongo) RemoveInstances(ctx context.Context, ids []string, owner string) error {
	filter := bson.M{ownerKey: owner, idKey: bson.M{"$in": ids}}
	_, err := this.instanceCollection().DeleteMany(ctx, filter)
	return err
}

func (this *Mongo) GetInstances(ctx context.Context, ids []string, owner string) (result []model.Instance, allExist bool, err error) {
	filter := bson.M{ownerKey: owner, idKey: bson.M{"$in": ids}}
	cursor, err := this.instanceCollection().Find(ctx, filter)
	if err != nil {
		return result, false, err
	}
	for cursor.Next(context.Background()) {
		instance := model.Instance{}
		err = cursor.Decode(&instance)
		if err != nil {
			return result, false, err
		}
		result = append(result, instance)
	}

	return result, len(result) == len(ids), nil
}


