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
	"log"
	"strings"

	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const idFieldName = "Id"
const nameFieldName = "Name"
const descriptionFieldName = "Description"
const entityNameFieldName = "EntityName"
const serviceNameFieldName = "ServiceName"
const userIdFieldName = "UserId"
const createdAtFieldName = "CreatedAt"
const updatedAtFieldName = "UpdatedAt"
const generatedFieldName = "Generated"

var idKey string
var nameKey string
var descriptionKey string
var entityNameKey string
var serviceNameKey string
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
	descriptionKey, err = getBsonFieldName(model.Instance{}, descriptionFieldName)
	if err != nil {
		log.Fatal(err)
	}
	entityNameKey, err = getBsonFieldName(model.Instance{}, entityNameFieldName)
	if err != nil {
		log.Fatal(err)
	}
	serviceNameKey, err = getBsonFieldName(model.Instance{}, serviceNameFieldName)
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

func (this *Mongo) GetInstance(ctx context.Context, id string) (instance model.Instance, exists bool, err error) {
	result := this.instanceCollection().FindOne(ctx, bson.M{idKey: id})
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

func (this *Mongo) ListInstances(ctx context.Context, limit int64, offset int64, sort string, asc bool, search string, includeGenerated bool, ids []string) (result []model.Instance, err error) {
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
	opt.SetSort(bson.D{{sortby, direction}})

	searchKey := nameKey
	searchSplit := strings.Split(search, ":")
	if len(searchSplit) > 1 {
		switch searchSplit[0] {
		case "id":
			searchKey = idKey
		case "name":
			searchKey = nameKey
		case "entity_name":
			searchKey = entityNameKey
		case "description":
			searchKey = descriptionKey
		case "service_name":
			searchKey = serviceNameKey
		}
		search = searchSplit[1]
	}
	var filter bson.M
	if includeGenerated {
		filter = bson.M{searchKey: primitive.Regex{
			Pattern: ".*" + search + ".*",
		}}
	} else {
		// filter for generatedKey == False || generatedKey == undefined to find legacy instances
		filter = bson.M{"$or": []bson.M{{generatedKey: false}, {generatedKey: bson.M{"$exists": false}}},
			searchKey: primitive.Regex{
				Pattern: ".*" + search + ".*",
			}}
	}
	if ids != nil {
		filter[idKey] = bson.M{"$in": ids}
	}
	cursor, err := this.instanceCollection().Find(ctx, filter, opt)
	if err != nil {
		return nil, err
	}
	for cursor.Next(context.Background()) {
		instance := model.Instance{}
		err = cursor.Decode(&instance)
		if err != nil {
			return nil, err
		}
		result = append(result, instance)
	}
	if cursor.Err() != nil {
		return nil, cursor.Err()
	}
	return
}

func (this *Mongo) SetInstance(ctx context.Context, instance model.Instance) error {
	_, err := this.instanceCollection().ReplaceOne(ctx, bson.M{idKey: instance.Id}, instance, options.Replace().SetUpsert(true))
	if err != nil {
		log.Println("Cant set instance to db: " + err.Error())
	}
	return err
}

func (this *Mongo) RemoveInstances(ctx context.Context, ids []string) error {
	filter := bson.M{idKey: bson.M{"$in": ids}}
	_, err := this.instanceCollection().DeleteMany(ctx, filter)
	return err
}

func (this *Mongo) GetInstances(ctx context.Context, ids []string) (result []model.Instance, allExist bool, err error) {
	filter := bson.M{idKey: bson.M{"$in": ids}}
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
