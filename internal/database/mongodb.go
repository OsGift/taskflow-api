package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/OsGift/taskflow-api/internal/models"
)

// ConnectMongoDB establishes a connection to MongoDB
func ConnectMongoDB(uri, dbName string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	// Ping the primary to verify connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	log.Printf("Successfully connected to MongoDB: %s", uri)
	return client, nil
}

// SeedDefaultRoles ensures that default roles exist in the database
func SeedDefaultRoles(db *mongo.Database) error {
	rolesCollection := db.Collection("roles")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, defaultRole := range models.DefaultRoles {
		filter := bson.M{"name": defaultRole.Name}
		var existingRole models.Role
		err := rolesCollection.FindOne(ctx, filter).Decode(&existingRole)

		if err == mongo.ErrNoDocuments {
			// Role does not exist, insert it
			_, err := rolesCollection.InsertOne(ctx, defaultRole)
			if err != nil {
				return err
			}
			log.Printf("Seeded default role: %s", defaultRole.Name)
		} else if err != nil {
			// Other error than not found
			return err
		} else {
			// Role exists, update its permissions to ensure they are current
			update := bson.M{
				"$set": bson.M{
					"permissions": defaultRole.Permissions,
				},
			}
			_, err = rolesCollection.UpdateOne(ctx, filter, update)
			if err != nil {
				return err
			}
			log.Printf("Updated existing default role: %s", defaultRole.Name)
		}
	}
	return nil
}
