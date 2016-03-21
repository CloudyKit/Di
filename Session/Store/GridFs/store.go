// mgoStore uses the mgo mongo driver to store sessions on mongo gridfs
package GridFs

import (
	"github.com/CloudyKit/framework/Session"
	"github.com/jhsx/qm"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"time"
)

// New returns a new store
func New(db, prefix string, mgoSess func() *mgo.Session) Session.Store {
	return &Store{
		db:      db,
		prefix:  prefix,
		session: mgoSess,
	}
}

type sessionCloser struct {
	session *mgo.Session
	*mgo.GridFile
}

func (ss *sessionCloser) Done() {
	if ss.GridFile != nil {
		if err := ss.GridFile.Close(); err != nil {
			ss.session.Close()
			return
		}
	}
	ss.session.Close()
}

type Store struct {
	session    func() *mgo.Session
	db, prefix string
}

func (sessionStore *Store) gridFs(name string, create bool) (*sessionCloser) {
	session := sessionStore.session()
	session.SetMode(mgo.Strong, false)
	gridFs := session.DB(sessionStore.db).GridFS(sessionStore.prefix)
	if create {
		gridFs.Remove(name)
		gridFile, err := gridFs.Create(name)
		if err != nil {
			panic(err)
		}
		return &sessionCloser{session: session, GridFile: gridFile}
	}
	gridFile, err := gridFs.Open(name)
	if err != nil {
		panic(err)
	}
	return &sessionCloser{session: session, GridFile: gridFile}
}

func (sessionStore *Store) Writer(name string) (writer io.WriteCloser) {
	return sessionStore.gridFs(name, true)
}

func (sessionStore *Store) Reader(name string) (reader io.ReadCloser) {
	return sessionStore.gridFs(name, false)
}

func (sessionStore *Store) Gc(before time.Time) {
	sess := sessionStore.session()
	defer sess.Close()

	gridFs := sess.DB(sessionStore.db).GridFS(sessionStore.prefix)

	var fileId struct {
		Id bson.ObjectId `bson:"_id"`
	}

	inter := gridFs.Find(qm.Lt("uploadDate", before)).Iter()
	defer inter.Close()

	for inter.Next(&fileId) {
		gridFs.RemoveId(fileId.Id)
	}

	return
}
