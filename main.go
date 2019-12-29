package main

import (
	"compress/gzip"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"time"
	"unsafe"

	"github.com/iotaledger/iota.go/trinary"
	cuckoo "github.com/seiflotfy/cuckoofilter"
)

var cf = cuckoo.NewFilter(100000000)
var spentAddrs = map[string]struct{}{}

var randomlyGeneratedAddrs = 10000000

// Converts a slice of bytes into a string without performing a copy.
// NOTE: This is an unsafe operation and may lead to problems if the bytes
// passed as argument are changed while the string is used.  No checking whether
// bytes are valid UTF-8 data is performed.
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Converts a string into a slice of bytes without performing a copy.
// NOTE: This is an unsafe operation and may lead to problems if the bytes are changed.
func StringToBytes(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	b := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len,
	}))
	// ensure the underlying string doesn't get GC'ed before the assignment happens
	runtime.KeepAlive(&s)

	return b
}

func main() {
	fmt.Println("loading spent addresses...")
	s := time.Now()
	if err := LoadSnapshotFromFile("./latest-export.gz.bin"); err != nil {
		panic(err)
	}
	fmt.Printf("took %v\n", time.Now().Sub(s))

	fmt.Println("serializing cuckoo filter to file...")
	s = time.Now()
	if err := ioutil.WriteFile("cuckoo.filter", cf.Encode(), os.ModePerm); err != nil {
		panic(err)
	}
	fmt.Printf("took %v\n", time.Now().Sub(s))

	/*
	fmt.Println("now looking up every spent address in the cuckoo filter")
	s = time.Now()
	var i int
	for spentAddr := range spentAddrs {
		if has := cf.Lookup(StringToBytes(spentAddr)); !has {
			panic(fmt.Sprintf("spent address at index %d was not in the cuckoo filter", i))
		}
		fmt.Printf("%d/%d\t\r", i, len(spentAddrs))
		i++
	}
	fmt.Println()
	fmt.Printf("took %v\n", time.Now().Sub(s))
	 */

	fmt.Printf("now looking up %d randomly generated spent addresses\n", randomlyGeneratedAddrs)
	s = time.Now()
	var falsePositive int
	var skippedDup int
	for i := 0; i < randomlyGeneratedAddrs; i++ {
		randAddr := make([]byte, 49)
		if _, err := rand.Read(randAddr); err != nil {
			panic(err)
		}
		if _, has := spentAddrs[BytesToString(randAddr)]; has {
			skippedDup++
			continue
		}
		if has := cf.Lookup(randAddr); has {
			falsePositive++
		}
		fmt.Printf("%d/%d/(falsePositive=%d)/(skippedDup=%d)\t\r", i, randomlyGeneratedAddrs, falsePositive, skippedDup)
	}
	fmt.Println()
	fmt.Printf("took %v\n", time.Now().Sub(s))
}

func LoadSnapshotFromFile(filePath string) error {
	fmt.Println("Loading snapshot file...")

	file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	hashBuf := make([]byte, 49)
	_, err = gzipReader.Read(hashBuf)
	if err != nil {
		return err
	}

	var msIndex int32
	var msTimestamp int64
	var solidEntryPointsCount, seenMilestonesCount, ledgerEntriesCount, spentAddrsCount int32

	_, err = trinary.BytesToTrytes(hashBuf)
	if err != nil {
		return err
	}

	err = binary.Read(gzipReader, binary.BigEndian, &msIndex)
	if err != nil {
		return err
	}

	err = binary.Read(gzipReader, binary.BigEndian, &msTimestamp)
	if err != nil {
		return err
	}

	err = binary.Read(gzipReader, binary.BigEndian, &solidEntryPointsCount)
	if err != nil {
		return err
	}

	err = binary.Read(gzipReader, binary.BigEndian, &seenMilestonesCount)
	if err != nil {
		return err
	}

	err = binary.Read(gzipReader, binary.BigEndian, &ledgerEntriesCount)
	if err != nil {
		return err
	}

	err = binary.Read(gzipReader, binary.BigEndian, &spentAddrsCount)
	if err != nil {
		return err
	}

	for i := 0; i < int(solidEntryPointsCount); i++ {
		var val int32

		err = binary.Read(gzipReader, binary.BigEndian, hashBuf)
		if err != nil {
			return fmt.Errorf("solidEntryPoints: %s", err)
		}

		err = binary.Read(gzipReader, binary.BigEndian, &val)
		if err != nil {
			return fmt.Errorf("solidEntryPoints: %s", err)
		}
	}

	for i := 0; i < int(seenMilestonesCount); i++ {

		var val int32

		err = binary.Read(gzipReader, binary.BigEndian, hashBuf)
		if err != nil {
			return fmt.Errorf("seenMilestones: %s", err)
		}

		err = binary.Read(gzipReader, binary.BigEndian, &val)
		if err != nil {
			return fmt.Errorf("seenMilestones: %s", err)
		}
	}

	ledgerState := make(map[trinary.Hash]uint64)
	for i := 0; i < int(ledgerEntriesCount); i++ {

		var val uint64

		err = binary.Read(gzipReader, binary.BigEndian, hashBuf)
		if err != nil {
			return fmt.Errorf("ledgerEntries: %s", err)
		}

		err = binary.Read(gzipReader, binary.BigEndian, &val)
		if err != nil {
			return fmt.Errorf("ledgerEntries: %s", err)
		}

		hash, err := trinary.BytesToTrytes(hashBuf)
		if err != nil {
			return fmt.Errorf("ledgerEntries: %s", err)
		}
		ledgerState[hash[:81]] = val
	}

	fmt.Printf("populating the cuckoo filter with %d spent addresses...\n", spentAddrsCount)
	for i := 0; i < int(spentAddrsCount); i++ {
		spentAddrBuf := make([]byte, 49)
		err = binary.Read(gzipReader, binary.BigEndian, spentAddrBuf)
		if err != nil {
			return fmt.Errorf("spentAddrs: %s", err)
		}
		if inserted := cf.Insert(spentAddrBuf); !inserted {
			panic("Cuckoo Filter capacity reached")
		}
		spentAddrs[BytesToString(spentAddrBuf[:])] = struct{}{}
	}

	fmt.Printf("populated the cuckoo filter with %d items\n", cf.Count())

	return nil
}
