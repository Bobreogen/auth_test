package main

import (
	"context"
	"encoding/base64"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
	"time"
)

type User struct {
	ID      primitive.ObjectID `bson:"_id" json:"id,omitempty"`
	GUID    string             `json:"guid"`
	Refresh string             `json:"refresh"`
	Access  string             `json:"access"`
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

var get = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("GUID") != "" {
		createTransaction(func(sessCtx mongo.SessionContext) (interface{}, error) {
			acStr, reStr, hashedPassword := getARH()
			collection := sessCtx.Client().Database("Users").Collection("users")
			_, _ = collection.DeleteMany(sessCtx, bson.M{"guid": r.FormValue("GUID")})
			_, _ = collection.InsertOne(sessCtx, User{primitive.NewObjectID(), r.FormValue("GUID"), base64.StdEncoding.EncodeToString(hashedPassword), acStr})
			_, _ = w.Write([]byte("access token: " + acStr + "\nrefresh token: " + reStr))
			return nil, nil
		})
	} else {
		_, _ = w.Write([]byte("Don't get a GUID"))
	}
})

var refr = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("refresh") != "" {
		createTransaction(func(sessCtx mongo.SessionContext) (interface{}, error) {
			if GUID := delByRefresh(w, r, sessCtx); GUID != "" {
				acStr, reStr, hashedPassword := getARH()
				_, _ = sessCtx.Client().Database("Users").Collection("users").
					InsertOne(sessCtx, User{primitive.NewObjectID(), GUID, base64.StdEncoding.EncodeToString(hashedPassword), acStr})
				_, _ = w.Write([]byte("access token: " + acStr + "\nrefresh token: " + reStr))
			}
			return nil, nil
		})
	} else {
		_, _ = w.Write([]byte("Don't get valid refresh token"))
	}
})

var delR = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("refresh") != "" {
		createTransaction(func(sessCtx mongo.SessionContext) (interface{}, error) {
			if GUID := delByRefresh(w, r, sessCtx); GUID != "" {
				_, _ = w.Write([]byte("Delete complete"))
			}
			return nil, nil
		})
	} else {
		_, _ = w.Write([]byte("Don't get valid refresh token"))
	}
})

func delByRefresh(w http.ResponseWriter, r *http.Request, sessCtx mongo.SessionContext) (GUID string) {
	var res []User
	collection := sessCtx.Client().Database("Users").Collection("users")
	cursor, _ := collection.Find(sessCtx, bson.M{})
	for cursor.Next(sessCtx) {
		var n User
		_ = cursor.Decode(&n)
		res = append(res, n)
	}
	bl := false
	for _, el := range res {
		dRefr, _ := base64.StdEncoding.DecodeString(el.Refresh)
		err := bcrypt.CompareHashAndPassword(dRefr, []byte(r.FormValue("refresh")))
		if err == nil {
			GUID = el.GUID
			_, _ = collection.DeleteOne(sessCtx, bson.M{"refresh": el.Refresh})
			bl = true
		}
	}
	if !bl {
		_, _ = w.Write([]byte("Don't find valid refresh token"))
	}
	return
}

var delGUID = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("GUID") != "" {
		createTransaction(func(sessCtx mongo.SessionContext) (interface{}, error) {
			_, _ = sessCtx.Client().Database("Users").Collection("users").
				DeleteMany(sessCtx, bson.M{"guid": r.FormValue("GUID")})
			_, _ = w.Write([]byte("Delete complete"))
			return nil, nil
		})
	} else {
		_, _ = w.Write([]byte("Don't get a GUID"))
	}
})

func createTransaction(fn func(sessCtx mongo.SessionContext) (interface{}, error)) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, _ := mongo.Connect(ctx, options.Client().ApplyURI("mongodb+srv://user:useruser@bobreogen.vmkfp.mongodb.net/admin?retryWrites=true&w=majority"))
	defer func() { _ = client.Disconnect(ctx) }()
	session, _ := client.StartSession()
	defer session.EndSession(ctx)
	_, _ = session.WithTransaction(ctx, fn)
}

func getARH() (acStr string, reStr string, hashedPassword []byte) {
	atClaims := jwt.MapClaims{}
	atClaims["exp"] = time.Now().Add(time.Hour * 240).Unix()
	access, refresh := jwt.New(jwt.SigningMethodHS512), jwt.NewWithClaims(jwt.SigningMethodHS512, atClaims)
	acStr, _ = access.SignedString([]byte("secret" + string(time.Now().Nanosecond())))
	reStr, _ = refresh.SignedString([]byte(string(time.Now().Nanosecond()) + "secret"))
	hashedPassword, _ = bcrypt.GenerateFromPassword([]byte(reStr), bcrypt.DefaultCost)
	return
}
