package archive

import (
	"errors"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"

	"github.com/JackalLabs/jackal-provider/jprov/types"
)

type ArchiveDB interface {
	GetFid(cid types.Cid) (fid types.Fid, found bool)
	GetContracts(fid types.Fid) (cid []types.Cid)
	SetContract(cid types.Cid, fid types.Fid) error
	DeleteContract(cid types.Cid)
	NewIterator() iterator.Iterator
	Close() error
} 

type DowntimeDB interface {
	Get(cid types.Cid) (blocks int)
	Set(cid types.Cid, blocks int)
	Delete(cid types.Cid)
	Close() error
}

var _ ArchiveDB = &DoubleRefArchiveDB{}


const cidSeparator = ","

type DoubleRefArchiveDB struct {
	db *leveldb.DB
}

func NewDoubleRefArchiveDB (filepath string) (*DoubleRefArchiveDB, error) {
	db, err := leveldb.OpenFile(filepath, nil)
	if err != nil {
		return nil, err
	}

	return &DoubleRefArchiveDB{db: db}, nil
}

func (d *DoubleRefArchiveDB) GetFid(cid types.Cid) (fid types.Fid, found bool) {
	value, err := d.db.Get(d.Key(cid), nil)
	if err != nil {
		return "", false
	}
	return types.Fid(value), true
}
func (d *DoubleRefArchiveDB) GetContracts(fid types.Fid) (cid []types.Cid){
	value, err := d.db.Get([]byte(fid), nil)
	if err != nil {
		panic(err)
	}
	cids := strings.Split(string(value), cidSeparator)
	for _, c := range cids {
		cid = append(cid, types.Cid(c))
	}
	return 
}
func (d *DoubleRefArchiveDB) SetContract(cid types.Cid, fid types.Fid) error {
	value, err := d.db.Get([]byte(cid), nil)
	if value != nil {
		return errors.New("already exist")
	}

	err = d.db.Put([]byte(cid), []byte(fid), nil)
	if err != nil {
		return err
	}

	err = d.setReference(cid, fid)
	return err
}

func (d *DoubleRefArchiveDB) setReference(cid types.Cid, fid types.Fid) error {
	value, err := d.db.Get([]byte(cid), nil)
	if err == leveldb.ErrNotFound {
		value = nil
	} else {
		return err
	}

	build := strings.Builder{}
	_, _ = build.WriteString(string(value))
	_, _ = build.WriteString(cidSeparator)
	_, _ = build.WriteString(string(cid))

	err = d.db.Put([]byte(fid), []byte(build.String()), nil)
	return err
}

func (d *DoubleRefArchiveDB) DeleteContract(cid types.Cid){
	err := d.db.Delete([]byte(cid), nil)
	if err != nil {
		panic(err)
	}
}
func (d *DoubleRefArchiveDB) NewIterator() iterator.Iterator{
	return d.db.NewIterator(nil, nil)
}
func (d *DoubleRefArchiveDB) Close() error{
	return d.db.Close()
}

func (d *DoubleRefArchiveDB) Key(cid types.Cid) (key []byte) {
	return []byte(cid)
}
