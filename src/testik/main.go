package main

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const MGO_COLLECTION = "testik"

var mgodb = newMgo("juser:jpass@ds031613.mongolab.com:31613/junodb")

func main() {
	return
}

func job(c *mgo.Collection) error {
	task := struct {
		ID bson.ObjectId `bson:"_id"`
	}{
		bson.NewObjectId(),
	}

	if err := c.Insert(task); err != nil {
		return err
	}

	if err := c.RemoveId(task.ID); err != nil {
		return err
	}
	return nil
}

func dbsync() (*mgo.Collection, func()) {
	sess := mgodb
	release := func() {
	}
	return sess.DB("").C(MGO_COLLECTION), release
}

func db() (*mgo.Collection, func()) {
	sess := mgodb.Copy()
	release := func() {
		sess.Close()
	}
	return sess.DB("").C(MGO_COLLECTION), release
}

func newMgo(url string) *mgo.Session {
	// connect to mongo
	sess, err := mgo.Dial(url)
	if err != nil {
		panic(err)
	}

	// Optional. Switch the session to a monotonic behavior.
	sess.SetMode(mgo.Monotonic, true)
	return sess

	c := sess.DB("").C(MGO_COLLECTION)
	// create indexes if don't exist
	indexes := []mgo.Index{
		// to control shop duplicates,
		mgo.Index{
			Key:        []string{"domain"},
			Unique:     true,
			DropDups:   false,
			Background: true,
			Sparse:     true,
		},
		// for fast stage access
		mgo.Index{
			Key:        []string{"stage"},
			DropDups:   false,
			Background: true,
			Sparse:     true,
		},
		/*
			// to get confirmed user
			mgo.Index{
				Key:        []string{"_id", "confirm"},
				Unique:     true,
				Background: true,
				Sparse:     true,
			},
		*/
	}

	for _, index := range indexes {
		if err := c.EnsureIndex(index); err != nil {
			panic(err)
		}
	}

	return sess
}
