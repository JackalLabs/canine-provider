package archive

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

type ArchiveDB interface {
	GetFid(cid string) (string, error)
	GetContracts(fid string) ([]string, error)
	SetContract(cid string, fid string) error
	DeleteContract(cid string) (purge bool, err error)
	NewIterator() iterator.Iterator
	Close() error
}

var _ ArchiveDB = &DoubleRefArchiveDB{}

const cidSeparator = ","

type DoubleRefArchiveDB struct {
	db *leveldb.DB
}

func NewDoubleRefArchiveDB(filepath string) (*DoubleRefArchiveDB, error) {
	db, err := leveldb.OpenFile(filepath, nil)
	if err != nil {
		return nil, err
	}

	return &DoubleRefArchiveDB{db: db}, nil
}

func (d *DoubleRefArchiveDB) GetFid(cid string) (string, error) {
	value, err := d.db.Get(d.key(cid), nil)
    if errors.Is(err, leveldb.ErrNotFound) {
        return "", ErrFidNotFound
    }
	if err != nil {
		return "", err
	}
	return string(value), err
}

func (d *DoubleRefArchiveDB) GetContracts(fid string) ([]string, error) {
	value, err := d.db.Get([]byte(fid), nil)
	if err != nil {
		return nil, err
	}
    return strings.Split(string(value), cidSeparator), nil
}

func (d *DoubleRefArchiveDB) SetContract(cid string, fid string) error {
    value, err := d.db.Get([]byte(cid), nil)//check if it already exists
    if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
        return err
    }
    if value != nil {
        return ErrContractAlreadyExists
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

func (d *DoubleRefArchiveDB) addReference(batch *leveldb.Batch, cid string, fid string) error {
	value, err := d.db.Get([]byte(fid), nil)
	if errors.Is(err, leveldb.ErrNotFound) {
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

func (d *DoubleRefArchiveDB) DeleteContract(cid string) (purge bool, err error) {
	batch := new(leveldb.Batch)
	purge, err = d.deleteReference(batch, cid)
	if err != nil {
		return
	}

	batch.Delete([]byte(cid))
	err = d.db.Write(batch, nil)
	return
}

func (d *DoubleRefArchiveDB) deleteReference(
	batch *leveldb.Batch,
	cid string,
) (purge bool, err error) {
	purge = false
	fid, err := d.db.Get([]byte(cid), nil)
	if err != nil {
		return
	}

	cidList, err := d.db.Get(fid, nil)
	if err != nil {
		return
	}

	var b strings.Builder
	b.WriteString(string(cid))
	b.WriteString(cidSeparator)

	result := strings.Replace(string(cidList), b.String(), "", 1)

	if len(result) == 0 {
		batch.Delete(fid)
		purge = true
	} else {
		batch.Put(fid, []byte(result))
	}

	return
}

func (d *DoubleRefArchiveDB) NewIterator() iterator.Iterator {
	return d.db.NewIterator(nil, nil)
}

func (d *DoubleRefArchiveDB) Close() error {
	return d.db.Close()
}

func (d *DoubleRefArchiveDB) key(cid string) (key []byte) {
	return []byte(cid)
}

func (d *DoubleRefArchiveDB) refKey(fid string) []byte {
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

func (d *DowntimeDB) NewIterator() iterator.Iterator {
	return d.db.NewIterator(nil, nil)
}

func (d *DowntimeDB) Get(cid string) (block int64, err error) {
	b, err := d.db.Get([]byte(cid), nil)
	if errors.Is(err, leveldb.ErrNotFound) {
		return 0, ErrContractNotFound
	}
	if err != nil {
		return
	}

	return ByteToBlock(b)
}

func (d *DowntimeDB) Set(cid string, block int64) error {
	b, err := BlockToByte(block)
	if err != nil {
		return err
	}
	return d.db.Put([]byte(cid), b, nil)
}

func (d *DowntimeDB) Delete(cid string) error {
	return d.db.Delete([]byte(cid), nil)
}

func (d *DowntimeDB) Close() error {
	return d.db.Close()
}

func ByteToBlock(b []byte) (int64, error) {
	r := bytes.NewReader(b)

	var block int64
	err := binary.Read(r, binary.LittleEndian, &block)
	return block, err
}

func BlockToByte(block int64) ([]byte, error) {
	b := new(bytes.Buffer)
	err := binary.Write(b, binary.LittleEndian, block)
	return b.Bytes(), err
}
