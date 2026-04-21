package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Address struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	ProvinceID  primitive.ObjectID `bson:"province_id,omitempty"`
	WardID      primitive.ObjectID `bson:"ward_id,omitempty"`
	Detail      string             `bson:"detail" json:"detail"`
	PhoneNumber string             `bson:"phone_number" json:"phone_number"`
}

type Province struct {
	ID    primitive.ObjectID `bson:"_id,omitempty"`
	Name  string             `bson:"name" json:"name"`
	Wards []Ward             `bson:"wards" json:"wards"`
}
type Ward struct {
	ID         string             `bson:"id" json:"id"`
	ProvinceID primitive.ObjectID `bson:"province_id,omitempty"`
	Name       string             `bson:"name" json:"name"`
}
