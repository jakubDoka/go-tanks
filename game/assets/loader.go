package assets

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	_ "image/png"

	"github.com/jakubDoka/goml/goss"
	"github.com/jakubDoka/mlok/ggl"
	"github.com/jakubDoka/mlok/ggl/drw"
	"github.com/jakubDoka/mlok/ggl/pck"
	"github.com/jakubDoka/mlok/ggl/txt"
	"github.com/jakubDoka/mlok/ggl/ui"
	"github.com/jakubDoka/mlok/load"
	"github.com/jakubDoka/mlok/mat"
	"github.com/jakubDoka/mlok/mat/rgba"
	"github.com/jakubDoka/sterr"
)

//go:generate genny -pkg=assets -in=$GOPATH\pkg\mod\github.com\jakub!doka\mlok@v0.3.8\logic\memory\ordered.go -out=gen-ordered.go gen "Key=string Value=Bullet,Tank,World"

//go:embed assets
var RawAssets embed.FS

var (
	ErrMapping = sterr.New("error when mapping directory tree")
	ErrConfig  = sterr.New("problem with a config file, add or fix config.json")
	ErrProblem = sterr.New("problem with a %s file on path %s")
	ErrFatal   = sterr.New("#ff0000[[fatal]]]")
	ErrWarming = sterr.New("#FFFF00[[note]]]")
)

type Assets struct {
	Stats
	GameConfig

	RawStats RawStats

	Loader, AppData load.Util
	Root            string
	Errors          []error

	ui.Assets
	UISources map[string]File
	UIScenes  map[string]*ui.Scene
	UIParser  ui.Parser

	Config

	buff []string
}

func NAssets() *Assets {
	dt, err := load.AppData("go-tanks")
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll(dt.Root, os.ModePerm)
	if err != nil {
		panic(err)
	}

	a := &Assets{
		Stats: NStats(),

		Assets: ui.Assets{
			Markdowns: map[string]*txt.Markdown{
				"default": txt.NMarkdown(),
			},
			Sheet:  pck.Sheet{Root: "textures"},
			Styles: goss.Styles{},
		},

		AppData: dt,

		UIParser:  *ui.NParser(),
		UISources: make(map[string]File),
	}

	a.LoadGameConfig()

	return a
}

func (a *Assets) Sprite(name string) ggl.Sprite {
	spr, ok := a.Sheet.Sprite(name)
	if !ok {
		spr, _ := a.Sheet.Sprite("does_not_exist")
		return spr
	}
	return spr
}

func (a *Assets) Compile() {
	for _, v := range a.Markdowns {
		a.Sheet.AddMarkdown(v)
	}
	a.Sheet.Pack()
	fmt.Println(a.Sheet.Regions)

	a.CompileStats()
	a.CompileUI()
}

func (a *Assets) CompileStats() {
	sv := reflect.ValueOf(&a.Stats).Elem()
	st := sv.Type()
	rv := reflect.ValueOf(a.RawStats)
	av := reflect.ValueOf(a)

	for i := 0; i < st.NumField(); i++ {
		stf := st.Field(i)
		svf := sv.Field(i)
		rvf := rv.FieldByName(stf.Name)

		put := svf.Addr().MethodByName("Put")
		proc := av.MethodByName(stf.Name[:len(stf.Name)-1])
		for k, v := range rvf.Interface().(goss.Styles) {
			name := reflect.ValueOf(k)
			put.Call([]reflect.Value{
				name,
				proc.Call([]reflect.Value{
					name,
					reflect.ValueOf(NStyle(v)),
				})[0],
			})
		}
	}
}

func (a *Assets) CompileUI() {
	a.UIParser.AddFactory("bar", &Bar{})

	a.UIScenes = make(map[string]*ui.Scene)
	for k, v := range a.UISources {
		scene := ui.NEmptyScene()
		scene.Assets = &a.Assets
		scene.Parser = &a.UIParser

		err := scene.Root.AddGoml(v.Data)
		if err != nil {
			a.Log(ErrProblem.Args("goml", v.Path).Wrap(err))
			continue
		}

		a.UIScenes[k] = scene
	}
}

func (a *Assets) Bullet(name string, stl RawStyle) Bullet {
	return Bullet{
		Speed:    stl.Float("speed", 500),
		Size:     stl.Float("size", 5),
		LiveTime: stl.Float("livetime", 1),
		Damage:   stl.Int("damage", 1),
		Sprite:   a.Sprite(stl.Ident("sprite", name+"3")),
	}
}

func (a *Assets) Tank(name string, stl RawStyle) Tank {
	return Tank{
		Bullet: a.Bullet(name, stl.Sub("bullet", a.RawStats.Bullets)),

		Speed:        stl.Float("speed", 2000),
		Transmission: stl.Float("transmission", .5),
		Steer:        stl.Float("steer_speed", 3),
		BaseSprite:   a.Sprite(stl.Ident("base_sprite", name+"2")),

		MaxHealth:         stl.Int("max_health", 50),
		Size:              stl.Float("size", 20),
		RetreatRatio:      stl.Int("retreat_ratio", 5),
		RegenerationProc:  stl.Float("regeneration_proc", 10),
		RegenerationTick:  stl.Float("regeneration_tick", 1),
		RegenerationPower: stl.Int("regeneration_power", 1),

		Next:        stl.Ident("next", ""),
		NeededScore: stl.Int("needed_score", 10),
		Value:       stl.Int("value", 1),

		Distancing:   stl.Float("distancing", .5),
		ReloadSpeed:  stl.Float("reload_speed", 1),
		TurretLen:    stl.Float("turret_len", 50),
		TurretSpeed:  stl.Float("turret_speed", 3),
		TurretSprite: a.Sprite(stl.Ident("turret_sprite", name+"1")),
		TurretPivot:  stl.Vec("turret_pivot", mat.V(-7, 0)),
		TurretOffset: stl.Vec("turret_offset", mat.V(-7, 0)),
		Memory:       stl.Float("memory", .5),
	}
}

func (a *Assets) World(name string, stl RawStyle) World {
	return World{
		Size:         stl.Vec("size", mat.V(5000, 5000)),
		Tile:         stl.Vec("tile_size", mat.V(300, 300)),
		Scale:        stl.Vec("scale", mat.V(1.5, 1.5)),
		Friction:     stl.Float("friction", 10),
		SpawnRate:    stl.Float("spawn_rate", 60),
		SpawnScaling: stl.Float("spawn_scaling", .6),
		TeamCount:    stl.Int("team_count", 2),
		Background:   stl.RGBA("background_color", rgba.Black),
		Spawns:       stl.IdentList("spawns"),
		Player:       stl.Ident("player", ""),
		WinMessage:   stl.Ident("win_message", "YOU WON!"),
		LoseMessage:  stl.Ident("lose_message", "YOU LOST!"),
	}
}

func (a *Assets) Load(root string, loader load.Loader) {
	a.Root = root
	a.Loader.Loader = loader

	a.LoadConfig()
	a.LoadTextures()
	a.LoadFonts()
	a.LoadUI()
	a.LoadStyles()
	a.VerifyStyles()
}

func (a *Assets) LoadGameConfig() {
	err := a.AppData.Json("config.json", &a.GameConfig)
	a.Log(err)
}

func (a *Assets) SaveGameConfig() error {
	bts, err := json.Marshal(a.GameConfig)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(a.AppData.Root, "config.json"), bts, os.ModePerm)
}

func (a *Assets) VerifyStyles() {

}

func (a *Assets) LoadStyles(styles ...string) {
	v := reflect.ValueOf(&a.RawStats).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		vf := v.Field(i)
		tf := t.Field(i)
		style := goss.Styles{}
		a.LoadStyle(a.Path("stats", strings.ToLower(tf.Name)), true, style)
		vf.Set(reflect.ValueOf(style))
	}
}

func (a *Assets) LoadUI() {
	a.LoadPrefabs()

	root := a.Path("ui")

	a.LoadStyle(root, false, a.Styles)

	list := a.ListPath(root, false, "goml")
	fmt.Println(list)
	for _, p := range list {
		bts, err := a.Loader.ReadFile(p)
		if err != nil {
			a.Log(err)
		}
		a.UISources[PathName(p)] = File{p, bts}
	}
}

func (a *Assets) LoadPrefabs() {
	list := a.ListPath(a.Path("ui", "prefabs"), false, "goml")
	for _, p := range list {
		bts, err := a.Loader.ReadFile(p)
		if err != nil {
			a.Log(err)
		}
		_, err = a.UIParser.GP.Parse(bts)
		a.Log(err)
	}
}

func (a *Assets) LoadConfig() {
	a.Config = Config{}
	err := a.Json(a.Path("config.json"), a.Config)
	a.Log(ErrConfig.Wrap(err))

}

func (a *Assets) LoadFonts() {
	list := a.ListPath(a.Path("fonts"), false, "ttf")
	m := a.Markdowns["default"]
	for _, p := range list {
		ttf, err := a.Loader.LoadTTF(p, a.FontSize)
		if err != nil {
			a.Log(ErrProblem.Args("font", p).Wrap(err))
			continue
		}

		atlas := txt.NAtlas(PathName(p), ttf, a.FontSpacing, txt.ASCII)
		m.Fonts[atlas.Name] = txt.NDrawer(atlas)
	}
}

func (a *Assets) LoadTextures() {
	list := a.ListPath(a.Path("textures"), true, "")
	for _, p := range list {
		image, err := a.Loader.LoadImage(p)
		if err != nil {
			a.Log(ErrProblem.Args("image", p).Wrap(err))
		}
		ggl.FlipNRGBA(image)
		a.Sheet.Data = append(a.Sheet.Data, pck.PicData{
			Img:  image,
			Name: p,
		})
	}
}

func (a *Assets) ListPath(root string, rec bool, ext string) []string {
	var err error
	a.buff, err = a.Loader.List(root, a.buff[:0], rec, ext)
	a.Log(ErrMapping.Wrap(err))
	return a.buff
}

func (a *Assets) Path(args ...string) string {
	a.buff = append(a.buff[:0], a.Root)
	a.buff = append(a.buff, args...)
	return path.Join(a.buff...)
}

// Json unmarshal-s json from given path to dest
func (a *Assets) Json(path string, dest interface{}) error {
	bts, err := a.Loader.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(bts, dest)
}

func (a *Assets) Log(err error) {
	a.AddErr(ErrWarming.Wrap(err))
}

func (a *Assets) Fatal(err error) {
	a.AddErr(ErrFatal.Wrap(err))
}

func (a *Assets) AddErr(err error) {
	if err != nil {
		a.Errors = append(a.Errors, err)
	}
}

func (a *Assets) LoadStyle(p string, rec bool, dest goss.Styles) {
	list := a.ListPath(p, rec, "goss")
	for _, p := range list {
		bts, err := a.Loader.ReadFile(p)
		if err != nil {
			a.Log(err)
			continue
		}

		style, err := a.UIParser.GS.Parse(bts)
		if err != nil {
			a.Log(ErrProblem.Args("goss", p).Wrap(err))
			continue
		}

		dest.Add(style)
	}
}

func PathName(p string) string {
	base := filepath.Base(p)
	if strings.Contains(base, ".") {
		base = base[:strings.LastIndex(base, ".")]
	}
	return base
}

type File struct {
	Path string
	Data []byte
}

type Stats struct {
	Bullets StringBulletOrdered
	Tanks   StringTankOrdered
	Worlds  StringWorldOrdered
}

func NStats() Stats {
	return Stats{
		Bullets: NStringBulletOrdered(),
		Worlds:  NStringWorldOrdered(),
		Tanks:   NStringTankOrdered(),
	}
}

type RawStats struct {
	Bullets, Tanks, Worlds goss.Styles
}

type Config struct {
	FontSize    float64 `json:"font_size"`
	FontSpacing int     `json:"font_spacing"`
}

type World struct {
	Scale, Size, Tile                 mat.Vec
	Friction, SpawnRate, SpawnScaling float64
	Seed                              int64
	TeamCount                         int
	Background                        mat.RGBA
	Player, WinMessage, LoseMessage   string
	Spawns                            []string
}

type Tank struct {
	Bullet Bullet

	Speed, Transmission, Steer, RegenerationProc, RegenerationTick float64
	BaseSprite                                                     ggl.Sprite

	Size                                       float64
	MaxHealth, RetreatRatio, RegenerationPower int

	Next               string
	NeededScore, Value int

	TurretLen, ReloadSpeed, TurretSpeed, Memory, Distancing float64
	TurretPivot, TurretOffset                               mat.Vec
	TurretSprite                                            ggl.Sprite
}

type Bullet struct {
	Speed, Size, LiveTime float64
	Damage                int
	Sprite                ggl.Sprite
}

func (b *Bullet) Range() float64 {
	return b.Speed * b.LiveTime
}

func (b *Bullet) Range2() float64 {
	return b.Speed * b.LiveTime * b.Speed * b.LiveTime
}

type RawStyle struct {
	load.RawStyle
}

func NStyle(style goss.Style) RawStyle {
	return RawStyle{RawStyle: load.RawStyle{Style: style}}
}

func (r RawStyle) Sub(key string, fallback goss.Styles) RawStyle {
	if i, ok := r.Style.Ident(key); ok {
		return NStyle(fallback[i])
	}
	return RawStyle{
		RawStyle: r.RawStyle.Sub(key, load.RawStyle{
			Style: fallback[key],
		}),
	}
}

func (r RawStyle) IdentList(key string) (res []string) {
	val, ok := r.Style[key]
	if !ok {
		return
	}

	res = make([]string, len(val))
	for i := range val {
		res[i] = fmt.Sprint(val[i])
	}

	return
}

type Bar struct {
	ui.ModuleBase
	Progress, Max float64
	ProgressColor mat.RGBA
}

func (b *Bar) New() ui.Module {
	return &Bar{}
}

func (b *Bar) Init(e *ui.Element) {
	b.ModuleBase.Init(e)
	b.Max = e.Float("max_progress", 1)
	b.ProgressColor = e.RGBA("progress_color", rgba.White)
}

func (b *Bar) Draw(t ggl.Target, c *drw.Geom) {
	b.ModuleBase.Draw(t, c)
	c.Restart()
	c.Color(b.ProgressColor).AABB(mat.AABB{
		Min: b.Frame.Min,
		Max: b.Frame.Max.Sub(mat.V(b.Frame.W()*(1-b.Progress/b.Max), 0)),
	})
	c.Fetch(t)
}

type GameConfig struct {
	Mods                []string
	UIColor, Background mat.RGBA
	ScrollSensitivity   float64
}
