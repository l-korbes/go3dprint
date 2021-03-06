package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"time"

	s "github.com/deadsy/sdfx/sdf"
	"github.com/gorilla/websocket"
)

type ConvertSVG struct {
	Width  string
	Height string
	SVGb64 string
}

type ReturnObject struct {
	TwoD    []byte `json:"TwoD"`
	TwoDPNG []byte `json:"TwoDPNG"`
	ThreeD  []byte `json:"ThreeD"`
}

type payload struct {
	content []byte
	err     error
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var sendThis payload

func main() {
	called := time.Unix(0, 1572006261124258405)
	now := time.Now()
	diff := now.Sub(called)
	fmt.Println("Mesh reload time:", diff.Round(time.Millisecond))

	port := "8080"
	if len(os.Args) > 2 {
		port = os.Args[2]
	}
	sendThis.content, sendThis.err = wrappedAndTidy()
	http.HandleFunc("/", serve)
	fmt.Println(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func serve(w http.ResponseWriter, r *http.Request) {
	if sendThis.err != nil {
		fmt.Println(sendThis.err.Error())
		http.Error(w, sendThis.err.Error(), http.StatusInternalServerError)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err = conn.WriteMessage(1, sendThis.content); err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("Content sent to: %s\n", conn.RemoteAddr())

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), string(msg))
		if string(msg) == "ping" {
			if err = conn.WriteMessage(msgType, []byte("pong")); err != nil {
				fmt.Println(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}

func wrappedAndTidy() ([]byte, error) {
	var err error
	var stlFile []byte
	fileSVG := "shape.svg"
	fileSTL := "mesh.stl"
	svg, stl := magic()
	s.RenderSTL(stl, 100, fileSTL)
	stlFile, err = ioutil.ReadFile(fileSTL)
	if err != nil {
		return nil, err
	}
	s.RenderSVG(svg, 60, fileSVG, "fill:none;stroke:#02f2b4;stroke-width:2px")
	svgFile, err := ioutil.ReadFile(fileSVG)
	if err != nil {
		return nil, err
	}
	svgString := string(svgFile[57:])
	svgFile = []byte(svgString)
	payload, err := json.Marshal(ReturnObject{TwoD: svgFile, ThreeD: stlFile})
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func magic() (s.SDF2, s.SDF3) {
	var output2d s.SDF2
	var output3d s.SDF3

	//
	// #######                   ###
	// #        #    #  #    #   ###
	// #        #    #  ##   #   ###
	// #####    #    #  # #  #    #
	// #        #    #  #  # #
	// #        #    #  #   ##   ###
	// #         ####   #    #   ###
	//

	// 1. Start here
	circle := s.Circle2D(20)
	coin := s.Extrude3D(circle, 10)
	output2d, output3d = circle, coin

	// 2. Then uncomment this
	square := s.Box2D(s.V2{10, 80}, 0)
	cube := s.Extrude3D(square, 30)
	cube = s.Transform3D(cube, s.Translate3d(s.V3{90, 0, 95}))
	output2d = s.Union2D(circle, square)
	output3d = s.Union3D(coin, cube)

	// 3. Then this
	coin = s.Transform3D(coin, s.RotateY(s.DtoR(90)))
	coin = s.Transform3D(coin, s.Translate3d(s.V3{30, 70, 30}))
	coin2 := s.Extrude3D(circle, 10)
	coin2 = s.Transform3D(coin2, s.RotateY(s.DtoR(90)))
	coin2 = s.Transform3D(coin2, s.Translate3d(s.V3{30, -70, 30}))
	output3d = s.Union3D(cube, coin, coin2)

	// 4. Now let's try some different stuff
	b := s.NewBezier()
	b.Add(0, 0).HandleFwd(s.DtoR(0), 150)
	b.Add(50, 150).Mid()
	b.Add(0, 300).HandleRev(s.DtoR(0), 150)
	b.Close()
	b2d := s.Polygon2D(b.Polygon().Vertices())
	output2d = b2d

	// 5. And in 3D
	b3d := s.Revolve3D(b2d)
	output3d = b3d

	// 6. And voilá, we have a gopher!
	cube = s.Transform3D(s.Extrude3D(s.Box2D(s.V2{20, 20}, 0), 30), s.Translate3d(s.V3{90, 0, 95}))
	q := bezierBlobs(25, 25, 50, 37, 65, 32, 30)
	w := bezierBlobs(25, 25, 50, 37, 65, -32, 30)
	e := bezierBlobs(13, 26, 26, 30, 90, 0, 70)
	r := bezierBlobs(10, 10, 20, 10, 86, -40, 37)
	t := bezierBlobs(10, 10, 20, 10, 86, 40, 43)
	y := bezierBlobs(5, 5, 13, 13, 105, 0, 70)
	gopher := s.Union3D(cube, coin, coin2, b3d, q, w, e, r, t, y)
	output3d = gopher

	// 7. Now let's make a dilator
	output2d, output3d = dilator()

	// 8. And one more thing!
	gopher = s.ScaleUniform3D(gopher, 0.1925)
	gopher = s.Transform3D(gopher, s.Translate3d(s.V3{0, 0, -235}))
	output3d = s.Union3D(gopher, output3d)

	return output2d, output3d
}

func bezierBlobs(mX, mY, fY, h, tX, tY, tZ float64) s.SDF3 {
	b := s.NewBezier()
	b.Add(0, 0).HandleFwd(s.DtoR(0), h)
	b.Add(mX, mY).Mid()
	b.Add(0, fY).HandleRev(s.DtoR(0), h)
	b.Close()
	output2d := s.Polygon2D(b.Polygon().Vertices())
	output3d := s.Revolve3D(output2d)
	output3d = s.Transform3D(output3d, s.Translate3d(s.V3{tX, tY, tZ}))
	return output3d
}

func dilator() (s.SDF2, s.SDF3) {
	d := s.NewBezier()
	length := 200.0
	radius := 34.0 / 2
	steps := 50.0
	step := radius / steps
	for x := 1.0; x < radius; x += step {
		// Crazy math!
		p1 := (x * x) * 0.3
		p2 := (x / math.Abs(radius-x)) * 7
		y := (p1 + p2) * 0.2
		// Crazy math!
		if y > length {
			y = length
		}
		d.Add(x, -y)
	}
	d.Add(0, -length)
	d.Close()
	output2d := s.Polygon2D(d.Polygon().Vertices())
	output3d := s.Revolve3D(output2d)
	return output2d, output3d
}
