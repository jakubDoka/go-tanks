package game

import (
	"fmt"
	"math"
	"strconv"

	"github.com/jakubDoka/mlok/ggl"
	"github.com/jakubDoka/mlok/ggl/drw"
	"github.com/jakubDoka/mlok/ggl/key"
	"github.com/jakubDoka/mlok/ggl/key/binding"
	"github.com/jakubDoka/mlok/ggl/ui"
	"github.com/jakubDoka/mlok/logic/ai"
	"github.com/jakubDoka/mlok/logic/spatial"
	"github.com/jakubDoka/mlok/logic/timer"
	"github.com/jakubDoka/mlok/mat"
	"github.com/jakubDoka/mlok/mat/angle"
	"github.com/jakubDoka/mlok/mat/lerp"
	"github.com/jakubDoka/mlok/mat/rgba"
	"github.com/jakubDoka/mlok/mat/rnd"
	"github.com/jakubDoka/tanks/game/assets"
)

//go:generate genny -pkg=game -in=$GOPATH\pkg\mod\github.com\jakub!doka\mlok@v0.3.7\logic\memory\storage.go -out=gen-storage.go gen "Element=Tank,Bullet"

type World struct {
	assets.World
	Original  *assets.World
	Spawning  timer.Timer
	GameState State

	Delta float64
	Frame mat.AABB

	*assets.Assets
	Batch ggl.Batch

	Tanks   TankStorage
	Bullets BulletStorage

	Hasher spatial.MinHash
	Drawer drw.Geom

	CamPos mat.Vec
	Zoom   float64

	UI ui.Processor

	Player, TotalScore int

	rnd.Rnd

	Buff []int
}

func NWorld(a *assets.Assets) *World {
	w := &World{
		Assets: a,
		Zoom:   1,
		Player: -1,
	}

	w.Batch.Texture = ggl.NTexture(w.Sheet.Pic, false)

	w.SetScene("main_menu")

	return w
}

func (w *World) LoadMap(singleplayer bool, world *assets.World) {
	w.Original = world
	w.World = *world
	w.TotalScore = 0

	size := w.Size.Div(w.Tile).Point()
	w.Hasher = spatial.NMinHash(size.X, size.Y, w.Tile)
	w.Spawning = timer.Period(w.SpawnRate)
	w.Rnd = rnd.Time()

	w.Drawer.Restart()
	w.Tanks.Clear()
	w.Bullets.Clear()

	if singleplayer {
		w.UIScenes["singleplayer"].ID("poppup").SetHidden(true)

		w.SetScene("singleplayer")
		t, _, ok := w.Assets.Tanks.Tank(w.World.Player)
		if ok {
			w.CreateTank(true, 0, mat.ZV, 0, 0, t)
		} else {
			w.RandomSpawn(true, 0)
		}

		w.UpdateScore()
		w.GameState = Singleplayer
	} else {

	}
}

func (w *World) SetScene(name string) {
	w.UI.SetScene(w.Assets.UIScenes[name])
}

func (w *World) Update(win *ggl.Window, delta float64) {
	w.Delta = math.Min(delta, .1)
	win.SetCamera(w.View())
	w.Frame = win.Rect()
	spc := mat.V(100, 100)
	w.Frame.Min.SubE(spc)
	w.Frame.Max.AddE(spc)

	if w.GameState != Menu {
		w.Batch.Clear()
		w.Drawer.Fetch(&w.Batch)
		w.Drawer.Clear()
		w.Drawer.Color(w.Background).AABB(w.Size.ToAABB())
		col := w.Background.Inverted()
		col.A = .1
		w.UpdatePlayer(win)

		for _, id := range w.Tanks.Occupied() {
			t := w.Tanks.Item(id)
			w.Drawer.Color(col)
			w.DrawTank(t)
			w.UpdateTank(t)
			w.ControlTank(t)

			if t.Dead() {
				w.Tanks.Remove(id)
				w.Hasher.Remove(t.Address, t.ID, t.Group)
			}
		}

		for _, id := range w.Bullets.Occupied() {
			b := w.Bullets.Item(id)

			w.DrawBullet(b)
			w.UpdateBullet(b)

			if b.Live.Done() {
				w.Bullets.Remove(id)
			}
		}

		w.Spawn()

		w.Batch.Draw(win)
	}

	w.Batch.Clear()
	w.UI.SetFrame(win.Frame())
	w.UI.Update(win, delta)
	w.UI.Fetch(&w.Batch)
	win.SetCamera(mat.IM)
	w.Batch.Draw(win)

	if w.GameState == Menu {
		win.Update()
		win.Clear(rgba.Black)
		return
	}

	win.Update()
	win.Clear(w.Background.Inverted())
}

func (w *World) Spawn() {
	if !w.Spawning.TickDoneReset(w.Delta) {
		return
	}

	w.RandomSpawn(false, 1+w.Intn(w.TeamCount))
}

func (w *World) RandomSpawn(player bool, group int) {
	choice, _, ok := w.Assets.Tanks.Tank(w.Spawns[w.Intn(len(w.Spawns))])
	if !ok {
		return
	}
	w.CreateTank(
		player,
		group,
		mat.V(w.Float64()*w.Size.X, w.Float64()*w.Size.Y),
		w.Float64()*angle.Pi2,
		w.Float64()*angle.Pi2,
		choice,
	)
}

func (w *World) UpdatePlayer(win *ggl.Window) {

	if w.Player == -1 {
		return
	}

	p := w.Tanks.Item(w.Player)

	if win.MouseScroll().Y != 0 {
		w.Zoom *= 1 + win.MouseScroll().Y*(w.Assets.ScrollSensitivity+.2)
		w.Zoom = mat.Clamp(w.Zoom, .5, 3)
	}

	p.Input.Update(win)
	w.CamPos = p.Pos.Inv()
	prj := w.View().Unproject(win.MousePos())
	p.Aim = prj
}

func (w *World) View() mat.Mat {
	return mat.IM.Move(w.CamPos).Scaled(mat.ZV, w.Zoom)
}

func (w *World) UpdateTank(t *Tank) {
	t.Reloader.Tick(w.Delta)
	if !t.HealInter.Done() {
		col := t.HealInter.Update(w.Delta)
		t.Mask.R = col
		t.Mask.B = col
	}
	if !t.HitInter.Done() {
		col := t.HitInter.Update(w.Delta)
		t.Mask.G = col
		t.Mask.B = col
	}
	t.Pos.AddE(t.Vel.Scaled(w.Delta))
	t.Vel.SubE(t.Vel.Scaled(mat.Clamp(w.Friction*w.Delta, 0, 1)))
	t.Heal(w.Delta)

	w.Hasher.Update(&t.Address, t.Pos, t.ID, t.Group)
}

func (w *World) DrawTank(t *Tank) {
	if !w.Frame.Contains(t.Pos) {
		return
	}

	p := w.Tanks.Item(w.Player)

	if mat.Square(t.Pos, t.Size).Contains(p.Aim) {
		t.BarInter.Reset()
	}

	w.DrawTile(t.Pos, t.Size)

	t.BaseSprite.Draw(&w.Batch, mat.M(t.Pos, w.Scale, t.BaseRot), t.Mask)
	t.TurretSprite.Draw(
		&w.Batch,
		mat.M(t.Pos.Add(t.TurretOffset.Rotated(t.BaseRot)), w.Scale, t.BaseRot+t.TurretRot),
		t.Mask,
	)

	if !t.BarInter.Done() {
		col := mat.Alpha(t.BarInter.Update(w.Delta))
		progress := float64(t.Health) / float64(t.MaxHealth) * math.Pi
		if progress == math.Pi {
			progress = 0
		}
		w.Drawer.Arc(progress, -progress).Color(col).Thickness(3).Circle(mat.C(t.Pos.X, t.Pos.Y, t.Size*1.5))
	}
}

func (w *World) DrawTile(pos mat.Vec, size float64) {

	sqr := mat.Square(pos, size)
	min := w.Hasher.Adr(sqr.Min)
	max := w.Hasher.Adr(sqr.Max).Add(mat.P(1, 1))
	for y := min.Y; y < max.Y; y++ {
		for x := min.X; x < max.X; x++ {
			org := mat.V(float64(x)*w.Tile.X, float64(y)*w.Tile.Y)
			w.Drawer.AABB(mat.AABB{Min: org, Max: org.Add(w.Tile)})
		}
	}

}

func (w *World) CreateTank(player bool, group int, pos mat.Vec, rot, trot float64, tank *assets.Tank) *Tank {
	t, id := w.Tanks.Allocate()

	t.Tank = tank
	t.Pos = pos
	t.BaseRot = rot
	t.TurretRot = trot
	t.Health = tank.MaxHealth
	t.Reloader = timer.Period(tank.ReloadSpeed)
	t.Input = Bindings.Clone()
	t.BaseSprite = tank.BaseSprite
	t.TurretSprite = tank.TurretSprite
	t.Group = group
	t.ID = id
	t.Target = -1
	t.Player = player
	t.Healing = timer.Period(tank.RegenerationProc)
	t.Mask = rgba.White

	t.HealInter.End = 1
	t.HitInter.End = 1
	t.BarInter.Start = .5
	t.HitInter.Timer = timer.Period(.2)
	t.HealInter.Timer = timer.Period(tank.RegenerationTick)
	t.BarInter.Timer = timer.Period(2)

	t.TurretSprite.SetPivot(t.TurretPivot)
	w.Hasher.Insert(&t.Address, t.Pos, t.ID, t.Group)

	if player {
		if w.Player != -1 && w.Tanks.Used(w.Player) {
			w.Tanks.Item(w.Player).Health = 0
		}
		w.Player = id
	}

	return t
}

func (w *World) ControlTank(t *Tank) {
	if !w.Size.ToAABB().Contains(t.Pos) {
		t.Vel.AddE(mat.Rad(t.BaseRot, t.Speed*w.Delta))
		t.BaseRot = angle.Turn(angle.Norm(t.BaseRot), t.Pos.To(w.Size.Scaled(.5)).Angle(), t.Steer*w.Delta)
		return
	}

	total := t.TurretRot + t.BaseRot
	dir := t.Pos.To(t.Aim).Angle()
	t.TurretRot = angle.Turn(angle.Norm(total), dir, t.TurretSpeed*w.Delta) - t.BaseRot

	if t.Input.Pressed(Shoot) && t.Reloader.DoneReset() {
		w.CreateBullet(
			t.Pos.Add(t.TurretOffset.Rotated(t.BaseRot)).Add(mat.Rad(total, t.TurretLen)),
			t.Vel, t.Group, t.ID, total, &t.Bullet,
		)
	}

	if t.Input.Pressed(Forward) {
		t.Vel.AddE(mat.Rad(t.BaseRot, t.Speed*w.Delta))
	} else if t.Input.Pressed(Back) {
		t.Vel.SubE(mat.Rad(t.BaseRot, t.Speed*t.Transmission*w.Delta))
	}

	if !t.Player {
		w.UpdateAI(t)
		return
	}

	if t.Input.Pressed(Left) {
		t.BaseRot += t.Steer * w.Delta
	} else if t.Input.Pressed(Right) {
		t.BaseRot -= t.Steer * w.Delta
	}
}

func (w *World) UpdateAI(t *Tank) {

	if t.Target == -1 {
		w.Buff = w.Hasher.Query(mat.Square(t.Pos, t.Bullet.Range()), w.Buff[:0], t.Group, false)
		var (
			final = -1
			dest  = math.MaxFloat64
		)
		for _, id := range w.Buff {
			o := w.Tanks.Item(id)
			d := t.Pos.To(o.Pos).Len2()
			if d < dest {
				final = id
				dest = d
			}
		}
		if final == -1 || dest > t.Bullet.Range2() {
			return
		}
		t.Target = final
	} else if !w.Tanks.Used(t.Target) {
		t.DeTarget()
		return
	}

	o := w.Tanks.Item(t.Target)
	dif := t.Pos.To(o.Pos)
	if dif.Len2() > t.Bullet.Range2()*(1+t.Memory) {
		t.DeTarget()
		return
	}

	if t.ShouldRetreat() || dif.Len() < t.Bullet.Range()*t.Distancing {
		dif = dif.Inv()
	}

	w.Buff = w.Hasher.Query(mat.Square(t.Pos, 0), w.Buff[:0], t.Group, true)
	for _, id := range w.Buff {
		o := w.Tanks.Item(id)
		if mat.Square(t.Pos, t.Size*2).Intersects(mat.Square(o.Pos, o.Size*2)) {
			dif.AddE(o.Pos.To(t.Pos).Normal().Scaled(dif.Len()))
			break
		}
	}

	t.BaseRot = angle.Turn(angle.Norm(t.BaseRot), dif.Angle(), t.Steer*w.Delta)

	t.Input[Forward].State = binding.Pressed

	var ok bool
	t.Aim, ok = ai.Predict(t.Pos, o.Pos, o.Vel, t.Bullet.Speed)
	if ok && t.Pos.To(t.Aim).Len2() <= t.Bullet.Range2() && math.Abs(angle.To(t.Pos.To(t.Aim).Angle(), angle.Norm(t.TurretRot+t.BaseRot))) < math.Abs(math.Atan(o.Size/dif.Len())) {
		t.Input[Shoot].State = binding.Pressed
	} else {
		t.Input[Shoot].State = binding.Released
	}
}

func (w *World) UpdateBullet(b *Bullet) {
	bounds := mat.Square(b.Pos, b.Size)
	w.Buff = w.Hasher.Query(bounds, w.Buff[:0], b.Group, false)
	for _, id := range w.Buff {
		t := w.Tanks.Item(id)
		if t.Dead() || !mat.C(t.Pos.X, t.Pos.Y, t.Size).Intersects(mat.C(b.Pos.X, b.Pos.Y, b.Size)) {
			continue
		}

		t.Hit(b)
		if t.Dead() {
			w.OnDeath(b.Owner, id)
		} else {
			b.Live.Skip()
		}
		return
	}

	b.Pos.AddE(mat.Rad(b.Rot, b.Speed*w.Delta))
	b.Live.Tick(w.Delta)
}

func (w *World) OnDeath(killer, victim int) {
	if !w.Tanks.Used(killer) {
		return
	}

	k := w.Tanks.Item(killer)
	v := w.Tanks.Item(victim)
	k.Score += v.Value
	if killer == w.Player {
		w.TotalScore += v.Value
		w.UpdateScore()
	}
	if k.Score >= k.NeededScore {
		w.LevelUp(killer)
	}

	if victim == w.Player {
		w.EndGame(false)
	}
}

func (w *World) UpdateScore() {
	t := w.Tanks.Item(w.Player)
	scene := w.UIScenes["singleplayer"]
	bar := scene.ID("bar").Module.(*assets.Bar)
	bar.Progress = float64(t.Score)
	bar.Max = float64(t.NeededScore)
	scene.ID("score").Module.(*ui.Text).SetText(fmt.Sprintf("%d/%d", t.Score, t.NeededScore))
	scene.Redraw.Notify()
}

func (w *World) LevelUp(id int) {
	t := w.Tanks.Item(id)
	next, _, ok := w.Assets.Tanks.Tank(t.Next)
	if !ok || (t.Player && w.DisabledPlayer[t.Next]) || w.DisabledEnemy[t.Next] {
		if t.Player {
			w.EndGame(true)
		}
		t.Score = 0
		return
	}
	if t.Player {
		w.World.SpawnRate *= w.World.SpawnScaling
	}
	w.CreateTank(t.Player, t.Group, t.Pos, t.BaseRot, t.TurretRot, next)
	t.Health = 0
}

func (w *World) EndGame(win bool) {
	w.Player = -1

	w.SetScene("end_screen")
	scene := w.UIScenes["end_screen"]
	message := scene.ID("message").Module.(*ui.Text)
	if win {
		message.SetText(w.WinMessage)
	} else {
		message.SetText(w.LoseMessage)
	}
	scene.ID("total").Module.(*ui.Text).SetText(strconv.Itoa(w.TotalScore))

	w.GameState = Menu
}

func (w *World) DrawBullet(b *Bullet) {
	if !w.Frame.Contains(b.Pos) {
		return
	}

	w.DrawTile(b.Pos, b.Size)

	b.Sprite.Draw(&w.Batch, mat.M(b.Pos, w.Scale, b.Rot), rgba.White)
}

func (w *World) CreateBullet(pos, vel mat.Vec, group, owner int, dir float64, bullet *assets.Bullet) {
	b, id := w.Bullets.Allocate()

	b.Bullet = bullet
	b.Pos = pos
	b.Rot = dir
	b.Live = timer.Period(bullet.LiveTime)
	b.Sprite = bullet.Sprite
	b.Group = group
	b.ID = id
	b.Owner = owner
}

const (
	Forward binding.B = iota
	Back
	Left
	Right

	Shoot
)

var Bindings = binding.New(
	key.W,
	key.S,
	key.A,
	key.D,

	key.MouseLeft,
)

type Tank struct {
	*assets.Tank
	Pos, Vel, Aim                 mat.Vec
	BaseRot, TurretRot            float64
	Reloader                      timer.Timer
	Health                        int
	Name                          string
	BaseSprite, TurretSprite      ggl.Sprite
	Input                         binding.S
	Player                        bool
	Address                       mat.Point
	Group, ID, Target, Score      int
	Healing                       timer.Timer
	Mask                          mat.RGBA
	HitInter, HealInter, BarInter Interpolator
}

func (t *Tank) Hit(b *Bullet) {
	t.Health -= b.Damage
	t.Target = b.Owner
	t.Healing.Progress = 0
	t.Healing.Period = t.RegenerationProc

	t.HitInter.Reset()
	t.BarInter.Reset()
}

func (t *Tank) Heal(delta float64) {
	if t.Healthy() {
		return
	}

	if t.Healing.TickDoneReset(delta) {
		switch t.Healing.Period {
		case t.RegenerationProc:
			t.Healing.Period = t.RegenerationTick
		case t.RegenerationTick:
			t.Health = mat.Mini(t.Health+t.RegenerationPower, t.MaxHealth)
			t.HealInter.Reset()
		}
	}
}

func (t *Tank) Dead() bool {
	return t.Health <= 0
}

func (t *Tank) Healthy() bool {
	return t.Health == t.MaxHealth
}

func (t *Tank) ShouldRetreat() bool {
	if t.Dead() {
		return false
	}
	return t.MaxHealth/t.Health > t.RetreatRatio
}

func (t *Tank) DeTarget() {
	t.Target = -1
	t.Input[Shoot].State = binding.Released
}

type Bullet struct {
	*assets.Bullet
	Pos              mat.Vec
	Rot              float64
	Live             timer.Timer
	Sprite           ggl.Sprite
	Group, ID, Owner int
}

type State uint8

const (
	Menu State = iota
	Singleplayer
	MultiplayerServer
	MultiplayerClient
)

type Interpolator struct {
	lerp.LinearTween
	timer.Timer
}

func (i *Interpolator) Update(delta float64) float64 {
	i.Timer.Tick(delta)
	if i.Done() {
		return i.End
	}
	return i.Value(i.Progress / i.Period)
}
