package utils

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/JackalLabs/jackal-provider/jprov/archive"

	"github.com/cosmos/cosmos-sdk/client"
)

// old version of tree path stores merkle trees at homeDir/storage/fid.tree
func GetOldTreePath(homeDir, fid string) string {
	fileName := fmt.Sprintf("%s.tree", fid)
	storageDir := filepath.Join(homeDir, "storage")
	return filepath.Join(storageDir, fileName)
}

func FindMigratedFile(ctx client.Context, fid string) (bool, error) {
	pathFactory := archive.NewSingleCellPathFactory(ctx.HomeDir)

	_, err := os.Stat(pathFactory.FilePath(fid))
	if err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	// we should never reach here because at this point file exist and not exist
	// manual investigation required at this point
	return false, fmt.Errorf("Error: FindMigratedFile: file exist and not exist at the same time: %s", err.Error())
}

func checkFileIntegrity(homeDir, matchFid string, file *os.File) (bool, error) {
	stat, err := file.Stat()
	if err != nil {
		err = errors.Join(errors.New("checkFileIntegrity: failed to get file info"), err)
		return false, err
	}

	checkTree, err := CreateMerkleTree(10240, stat.Size(), file, file)

	archive := archive.NewSingleCellArchive(homeDir)
	mtree, err := archive.RetrieveTree(matchFid)
	if err != nil {
		err = errors.Join(errors.New("checkFileIntegrity: failed to find merkel tree"), err)
		return false, err
	}

	originalRoot := hex.EncodeToString(mtree.Root())
	checkRoot := hex.EncodeToString(checkTree.Root())

	return originalRoot == checkRoot, nil
}

func postGlueCheck(homeDir, fid string) (bool, error) {
	pathFactory := archive.NewSingleCellPathFactory(homeDir)
	file, err := os.Open(pathFactory.FilePath(fid))
	if err != nil {
		return false, err
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	pass, err := checkFileIntegrity(homeDir, fid, file)
	return pass, err
}

func findOldMerkleTree(homedir, fid string) (bool, error) {
	oldTreePath := GetOldTreePath(homedir, fid)

	_, err := os.Stat(oldTreePath)
	if err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	// we should never reach here because at this point file exist and not exist
	// manual investigation required at this point
	return false, fmt.Errorf("Error: FindMigratedFile: file exist and not exist at the same time: %s", err.Error())
}

func fixOutOfPlaceMerkleTreeFile(homedir, fid string) error {
	pathFactory := archive.NewSingleCellPathFactory(homedir)
	oldLocation := oldMerkleTreePath(homedir, fid)

	return os.Rename(oldLocation, pathFactory.TreePath(fid))
}

func handleGlueingProcess(ctx client.Context, fid string) {
	fmt.Printf("Glueing blocks... ")
	alreadyMigrated, err := FindMigratedFile(ctx, fid)
	if err != nil {
		fmt.Printf("Error: Migrate: %s", err.Error())
		alreadyMigrated = true // skip glueing
	}
	if !alreadyMigrated {
		err = GlueAllBlocks(ctx.HomeDir, fid)
		if err != nil {
			fmt.Printf("Error: Glueing failed: %s", err)
		}
	}
	fmt.Printf("done\n")
}

func handleFileIntegrityCheckProcess(ctx client.Context, fid string) {
	fmt.Printf("File integrity check... ")
	pass, err := postGlueCheck(ctx.HomeDir, fid)
	if err != nil {
		fmt.Printf("Error: Migrate: error during file integrity check: %s", err)
	}

	if !pass {
		fmt.Printf("File integrity check for %s failed! Delete the file or get it from other providers\n", fid)
	} else {
		fmt.Printf("passed\n")

		fmt.Printf("Cleaning old blocks... ")
		err = cleanOld(ctx.HomeDir, fid)
		if err != nil {
			fmt.Printf("Error: failed to clean old blocks, manual cleaning required: %s\n", err.Error())
		} else {
			fmt.Printf("done\n")
		}
	}
}

func handleMerkleTreeProcess(ctx client.Context, fid string) {
	fmt.Printf("Looking for old merkle tree... ")
	found, err := findOldMerkleTree(ctx.HomeDir, fid)
	if err != nil {
		fmt.Printf("Error: Migrate: error during search of old merkle tree: %s\n", err.Error())
	}

	if found {
		fmt.Printf("found\n")
		fmt.Printf("Moving old tree to new location... ")
		err = fixOutOfPlaceMerkleTreeFile(ctx.HomeDir, fid)
		if err != nil {
			fmt.Printf("Error: Migrate: failed to move the tree to new location: %s", err.Error())
		} else {
			fmt.Printf("done\n")
		}
	} else {
		fmt.Printf("not found\n")
	}
}

func Migrate(ctx client.Context) {
	fids, err := DiscoverFids(ctx.HomeDir)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		return
	}

	for _, fid := range fids {
		fmt.Printf("Migrating %s\n", fid)
		handleGlueingProcess(ctx, fid)
		handleMerkleTreeProcess(ctx, fid)
		handleFileIntegrityCheckProcess(ctx, fid)
		fmt.Printf("\n")
	}

	fmt.Printf("\n")
	fmt.Println("Migration finished")
}

func cleanOld(homeDir, fid string) error {
	pathFactory := archive.NewSingleCellPathFactory(homeDir)
	fileNames, err := GetBlockFileNames(pathFactory.FileDir(fid))
	if err != nil {
		return fmt.Errorf("failed to get block file names: %s", err)
	}
	for _, f := range fileNames {
		fullPath := filepath.Join(pathFactory.FileDir(fid), f)
		err = os.Remove(fullPath)
		if err != nil {
			return fmt.Errorf("failed to remove old blocks: %s", err)
		}
	}

	return nil
}

// DiscoverFids reads all directory entry of the storage and returns fids
func DiscoverFids(homeDir string) (fids []string, err error) {
	dirs, err := os.ReadDir(getStorageRootDir(homeDir))
	if err != nil {
		return
	}

	for _, dir := range dirs {
		if dir.IsDir() {
			fids = append(fids, dir.Name())
		}
	}

	return
}

// Glue all blocks for single fid
func GlueAllBlocks(homeDir, fid string) error {
	fileNames, err := GetBlockFileNames(getStoragePath(homeDir, fid))
	if err != nil {
		return err
	}

	err = glueAllBlocks(homeDir, fid, len(fileNames))

	return err
}

// glue all blocks starting from 0.jkl to <blocksCount>.jkl
func glueAllBlocks(homeDir, fid string, blockCount int) (err error) {
	pathFactory := archive.NewSingleCellPathFactory(homeDir)

	f, err := os.Create(pathFactory.FilePath(fid))
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	// glue files in order
	for i := 0; i < blockCount; i++ {
		path := filepath.Join(GetFidDir(homeDir, fid), getBlockFileName(i))
		if err := combine(f, path); err != nil {
			return err
		}
	}

	return
}

// Get all files' name in directory
// An error is returned if the directory contains more directory
func GetBlockFileNames(dir string) (fileNames []string, err error) {
	dirEntry, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, d := range dirEntry {
		if d.IsDir() {
			err = errors.New("this directory have another directory")
			return
		}

		ok := checkFileName(d.Name())
		if ok {
			fileNames = append(fileNames, d.Name())
		}
	}

	return
}

// combine opens source file and copy its contents into destination
func combine(dst io.Writer, srcFileName string) error {
	src, err := os.Open(srcFileName)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, src.Close())
	}()

	_, err = io.Copy(dst, src)
	return err
}

// Check if the file name is a valid block file name
// The fileName format should be in: i.jkl where i is an index
func checkFileName(filename string) bool {
	strIndex, ok := strings.CutSuffix(filename, ".jkl")
	if !ok {
		return ok
	}

	_, err := strconv.Atoi(strIndex)
	return err == nil
}

// Legacy file paths
func getStoragePath(homeDir, fid string) string {
	configPath := filepath.Join(homeDir, "storage")
	configFilePath := filepath.Join(configPath, fid)

	return configFilePath
}

func oldMerkleTreePath(homeDir, fid string) string {
	configPath := filepath.Join(homeDir, "storage")
	configTreePath := filepath.Join(configPath, fid+".tree")

	return configTreePath
}

// create file name for a block
func getBlockFileName(index int) string {
	var name strings.Builder
	_, _ = name.WriteString(strconv.Itoa(index)) // returns length of s and a nil err
	_, _ = name.WriteString(".jkl")              // returns length of s and a nil err

	return name.String()
}
