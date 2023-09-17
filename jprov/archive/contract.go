package archive

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"

	"github.com/JackalLabs/jackal-provider/jprov/types"
)

type ArchiveDB interface {
	GetFid(cid types.Cid) (types.Fid, error)
	GetContracts(fid types.Fid) ([]types.Cid, error)
	SetContract(cid types.Cid, fid types.Fid) error
	DeleteContract(cid types.Cid) error
	NewIterator() iterator.Iterator
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

func (d *DoubleRefArchiveDB) GetFid(cid types.Cid) (types.Fid, error) {
	value, err := d.db.Get(d.key(cid), nil)
	if err != nil {
		return "", err
	}
	return types.Fid(value), err
}

func (d *DoubleRefArchiveDB) GetContracts(fid types.Fid) ([]types.Cid, error){
	value, err := d.db.Get([]byte(fid), nil)
	if err != nil {
		return nil, err
	}
	var cid []types.Cid
	cids := strings.Split(string(value), cidSeparator)
	for _, c := range cids {
		cid = append(cid, types.Cid(c))
	}
	return cid, nil
}

func (d *DoubleRefArchiveDB) SetContract(cid types.Cid, fid types.Fid) error {
	value, err := d.db.Get([]byte(cid), nil)
	if err != nil {
		return err
	}
	if value != nil {
		return errors.New("already exist")
	}

	batch := new(leveldb.Batch)
	batch.Put([]byte(cid), []byte(fid))

	err = d.addReference(batch, cid, fid)
	if err != nil {
		return err
	}
	err = d.db.Write(batch, nil)
	return err
}

func (d *DoubleRefArchiveDB) addReference(batch *leveldb.Batch, cid types.Cid, fid types.Fid) error {
	value, err := d.db.Get([]byte(cid), nil)
	if err == leveldb.ErrNotFound {
		value = nil
	} else if err != nil {
		return err
	}

	// reference look like this "potato,tomato,...,onion,"
	var b strings.Builder
	_, _ = b.WriteString(string(value))
	_, _ = b.WriteString(string(cid))
	_, _ = b.WriteString(cidSeparator)

	batch.Put([]byte(fid), []byte(b.String()))
	return nil
}

func (d *DoubleRefArchiveDB) DeleteContract(cid types.Cid) error {
	batch := new(leveldb.Batch)
	err := d.deleteReference(batch, cid)
	if err != nil {
		return err
	}

	batch.Delete([]byte(cid))
	err = d.db.Write(batch, nil)
	return err
}

func (d *DoubleRefArchiveDB) deleteReference (batch *leveldb.Batch, cid types.Cid) error {
	fid, err := d.db.Get([]byte(cid), nil)
	if err != nil {
		return err
	}

	cidList, err := d.db.Get(fid, nil)
	if err != nil {
		return err
	}
	
	var b strings.Builder
	b.WriteString(string(cid))
	b.WriteString(cidSeparator)
	
	result := strings.Replace(string(cidList), b.String(), "", 1)

	if len(result) == 0 {
		batch.Delete(fid)
	} else {
		batch.Put(fid, []byte(result))
	}

	return nil
}

func (d *DoubleRefArchiveDB) NewIterator() iterator.Iterator{
	return d.db.NewIterator(nil, nil)
}

func (d *DoubleRefArchiveDB) Close() error{
	return d.db.Close()
}

func (d *DoubleRefArchiveDB) key(cid types.Cid) (key []byte) {
	return []byte(cid)
}

func (d *DoubleRefArchiveDB) refKey(fid types.Fid) []byte {
	return []byte(fid)
}

type DowntimeDB struct {
    db *leveldb.DB
}

func NewDowntimeDB(filepath string) (*DowntimeDB, error) {
    db, err := leveldb.OpenFile(filepath, nil)
    if err != nil {
        return nil, err
    }
    return &DowntimeDB{db: db}, nil
}

func (d *DowntimeDB) Get(cid types.Cid) (block int64, err error) {
    b, err := d.db.Get([]byte(cid), nil)
    block, err = byteToBlock(b)
    return
}

func (d *DowntimeDB) Set(cid types.Cid, block int64) error {
    b, err := blockToByte(block)
    if err != nil {
        return err
    }
    return d.db.Put([]byte(cid), b, nil)
}

func (d *DowntimeDB) Delete(cid types.Cid) error {
    return d.db.Delete([]byte(cid), nil)
}
func (d *DowntimeDB) Close() error {
    return d.db.Close()
}
func byteToBlock(b []byte) (int64, error) {
    r := bytes.NewReader(b)

    var block int64
    err := binary.Read(r, binary.LittleEndian, &block)
    return block, err
}
func blockToByte(block int64) ([]byte, error) {
    b := new(bytes.Buffer)
    err := binary.Write(b, binary.LittleEndian, block)
    return b.Bytes(), err
}
