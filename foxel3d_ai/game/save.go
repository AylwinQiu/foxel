// 存档：saves/<世界名>/world.foxel，gzip压缩的gob。
//
// 只保存改动过的chunk，没改过的下次重新生成（同一个种子生成的一样）。
// 方块按名字保存，所以mod增删了方块、注册顺序变了都能读；找不到的方块变成空气。
package game

import (
	"compress/gzip"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"foxel3d_ai/block"
	"foxel3d_ai/world"
)

const SAVE_VERSION = 1

type save_file struct {
	Version     int
	Seed        int64
	Player      Player
	Hotbar      []string
	Selected    int
	Build_depth int
	// 存档里BlockID i对应的方块名字
	Palette  []string
	Chunks   map[world.CubeID][]byte
	Fields   map[string]map[world.CubeID]string
	Mod_data map[string]string
}

func (g *Game) Save_path() string {
	return filepath.Join("saves", g.Config.World_name, "world.foxel")
}

// 存档。先写到临时文件再改名，写到一半出错不会弄坏原来的存档
func (g *Game) Save() error {
	for _, m := range mods {
		if m.On_save != nil {
			m.On_save(g)
		}
	}
	s := save_file{
		Version:     SAVE_VERSION,
		Seed:        g.Config.Seed,
		Player:      g.Player,
		Selected:    g.Selected,
		Build_depth: int(g.Build_depth),
		Chunks:      g.World.Saved_chunks(),
		Fields:      g.World.Fields(),
		Mod_data:    g.Mod_data,
	}
	for _, b := range g.Hotbar {
		s.Hotbar = append(s.Hotbar, block.Get(b).Name)
	}
	for _, b := range block.All() {
		s.Palette = append(s.Palette, block.Get(b).Name)
	}

	path := g.Save_path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	z := gzip.NewWriter(f)
	err = gob.NewEncoder(z).Encode(&s)
	if e := z.Close(); err == nil {
		err = e
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	g.last_save = g.Time
	return os.Rename(tmp, path)
}

// 没有存档时返回nil, nil
func read_save(path string) (*save_file, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("读存档%s失败：%w", path, err)
	}
	var s save_file
	if err := gob.NewDecoder(z).Decode(&s); err != nil {
		return nil, fmt.Errorf("读存档%s失败：%w", path, err)
	}
	if s.Version != SAVE_VERSION {
		return nil, fmt.Errorf("存档%s的版本是%d，现在只能读版本%d", path, s.Version, SAVE_VERSION)
	}
	return &s, nil
}

func (g *Game) apply_save(s *save_file) {
	ids := make([]world.BlockID, len(s.Palette))
	for i, name := range s.Palette {
		id, ok := block.By_name(name)
		if !ok {
			fmt.Fprintf(os.Stderr, "foxel: 存档里的方块%q没有注册（mod被删掉了？），变成空气\n", name)
		}
		ids[i] = id
	}
	remap := func(b world.BlockID) world.BlockID {
		if int(b) < len(ids) {
			return ids[b]
		}
		return world.AIR
	}
	if n := g.World.Set_saved_chunks(s.Chunks, remap); n > 0 {
		fmt.Fprintf(os.Stderr, "foxel: 存档里有%d个chunk坏了，重新生成\n", n)
	}
	g.World.Set_fields(s.Fields)
	g.Player = s.Player
	g.Selected = min(max(s.Selected, 0), HOTBAR_SIZE-1)
	g.Build_depth = world.Depth(min(max(s.Build_depth, int(world.MIN_DEPTH)), 0))
	for i, name := range s.Hotbar {
		if i < HOTBAR_SIZE {
			g.Hotbar[i], _ = block.By_name(name)
		}
	}
	if s.Mod_data != nil {
		g.Mod_data = s.Mod_data
	}
}
