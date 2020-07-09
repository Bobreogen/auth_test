package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"os"
	"time"
)

type User struct {
	ID primitive.ObjectID `bson:"_id" json:"id,omitempty"`
	GUID string `json:"guid"`
	Refresh string `json:"refresh"`
	Access string `json:"access"`

}

func main() {
	port := os.Getenv("PORT")
	r := mux.NewRouter()

	r.Handle("/", http.FileServer(http.Dir("./views/")))

	r.Handle("/get-tokens", get).Methods("GET")
	r.Handle("/refresh", refr).Methods("POST")
	r.Handle("/delete-token", delR).Methods("DELETE")
	r.Handle("/delete-tokens", delGUID).Methods("DELETE")

	_ = http.ListenAndServe(":"+port, r)

}

var get = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
	if r.FormValue("GUID") != "" {

		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		acStr, reStr := getAR()
		collection := getCollection()
		defer func() { _ = collection.Database().Client().Disconnect(ctx) }()

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(reStr), bcrypt.DefaultCost)

		callback := func(sessCtx mongo.SessionContext) (interface{}, error) {

			filter := bson.M{"guid": r.FormValue("GUID")}
			_, _ = collection.DeleteMany(context.TODO(), filter)
			result := User{primitive.NewObjectID(), r.FormValue("GUID"), base64.StdEncoding.EncodeToString(hashedPassword), acStr}
			_, _ = collection.InsertOne(context.TODO(), result)


			return nil, nil
		}
		session, err := collection.Database().Client().StartSession()
		if err != nil {
			panic(err)
		}
		defer session.EndSession(ctx)

		_, err = session.WithTransaction(ctx, callback)
		if err != nil {
			panic(err)
		}
		_, _ = w.Write([]byte("access token: " + acStr + "\nrefresh token: " + reStr))
	} else {
		_, _ = w.Write([]byte("Don't get a GUID"))
	}
})

var refr = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
	if r.FormValue("refresh") != ""{
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		collection := getCollection()
		defer func() { _ = collection.Database().Client().Disconnect(ctx) }()

		callback := func(sessCtx mongo.SessionContext) (interface{}, error) {

			var res []User
			cursor, err := collection.Find(ctx, bson.M{})
			if err != nil {
				fmt.Println(err)
			}
			for cursor.Next(ctx) {
				var n User
				_ = cursor.Decode(&n)
				res = append(res, n)
			}
			bl := false
			for _, el := range res {
				dRefr, _ := base64.StdEncoding.DecodeString(el.Refresh)
				err = bcrypt.CompareHashAndPassword( dRefr , []byte(r.FormValue("refresh")))
				if err == nil {
					_, _ = collection.DeleteOne(ctx, bson.M{"refresh": el.Refresh})
					acStr, reStr := getAR()
					hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(reStr), bcrypt.DefaultCost)
					result := User{primitive.NewObjectID(), el.GUID, base64.StdEncoding.EncodeToString(hashedPassword), acStr}
					_, _ = collection.InsertOne(context.TODO(), result)
					_, _ = w.Write([]byte("access token: " + acStr + "\nrefresh token: " + reStr))
					bl = true
				}
			}
			if !bl {
				_, _ = w.Write([]byte("Don't find valid refresh token"))
			}
			return nil, nil
		}
		session, err := collection.Database().Client().StartSession()
		if err != nil {
			panic(err)
		}
		defer session.EndSession(ctx)

		_, err = session.WithTransaction(ctx, callback)
		if err != nil {
			panic(err)
		}
	} else {
		_, _ = w.Write([]byte("Don't get valid refresh token"))
	}
})

var delR = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
	if r.FormValue("refresh") != ""{
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		collection := getCollection()
		defer func() { _ = collection.Database().Client().Disconnect(ctx) }()

		callback := func(sessCtx mongo.SessionContext) (interface{}, error) {

			var res []User
			cursor, err := collection.Find(ctx, bson.M{})
			if err != nil {
				fmt.Println(err)
			}
			for cursor.Next(ctx) {
				var n User
				_ = cursor.Decode(&n)
				res = append(res, n)
			}
			bl := false
			for _, el := range res {
				dRefr, _ := base64.StdEncoding.DecodeString(el.Refresh)
				err = bcrypt.CompareHashAndPassword( dRefr , []byte(r.FormValue("refresh")))
				if err == nil {
					_, _ = collection.DeleteOne(ctx, bson.M{"refresh": el.Refresh})
					_, _ = w.Write([]byte("Delete complete"))
					bl = true
				}
			}
			if !bl {
				_, _ = w.Write([]byte("Don't find valid refresh token"))
			}
			return nil, nil
		}
		session, err := collection.Database().Client().StartSession()
		if err != nil {
			panic(err)
		}
		defer session.EndSession(ctx)

		_, err = session.WithTransaction(ctx, callback)
		if err != nil {
			panic(err)
		}
	} else {
		_, _ = w.Write([]byte("Don't get valid refresh token"))
	}
})

var delGUID = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("GUID") != "" {

		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		collection := getCollection()
		defer func() { _ = collection.Database().Client().Disconnect(ctx) }()

		callback := func(sessCtx mongo.SessionContext) (interface{}, error) {

			filter := bson.M{"guid": r.FormValue("GUID")}
			_, _ = collection.DeleteMany(context.TODO(), filter)

			_, _ = w.Write([]byte("Delete complete"))
			return nil, nil
		}
		session, err := collection.Database().Client().StartSession()
		if err != nil {
			panic(err)
		}
		defer session.EndSession(ctx)

		_, err = session.WithTransaction(ctx, callback)
		if err != nil {
			panic(err)
		}
	} else {
		_, _ = w.Write([]byte("Don't get a GUID"))
	}
})

func getAR() (acStr string, reStr string){
	atClaims := jwt.MapClaims{}
	atClaims["exp"] = time.Now().Add(time.Hour * 240).Unix()
	access := jwt.New(jwt.SigningMethodHS512)
	acStr, err := access.SignedString([]byte("secret" + string(time.Now().Nanosecond())))
	if err != nil {
		log.Fatal(err)
	}
	refresh := jwt.NewWithClaims(jwt.SigningMethodHS512, atClaims)
	reStr, err = refresh.SignedString([]byte(string(time.Now().Nanosecond()) + "secret"))
	if err != nil {
		log.Fatal(err)
	}
	return
}

func getCollection() *mongo.Collection {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, _ := mongo.Connect(ctx, options.Client().ApplyURI("mongodb+srv://user:useruser@bobreogen.vmkfp.mongodb.net/admin?retryWrites=true&w=majority"))
	collection := client.Database("Users").Collection("users")
	return collection
}