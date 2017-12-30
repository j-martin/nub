package main

import (
	"github.com/paetzke/godot/godot"
	"log"
	"os"
	"strings"
)

func generateGraphs() {
	outputPath := "output"
	dirExist, err := PathExists(outputPath)
	if err != nil {
		log.Fatal(err)
	}
	if !dirExist {
		os.Mkdir(outputPath, os.FileMode(0775))
	}
	os.Chdir(outputPath)
	manifests := GetManifestRepository().GetAllManifests()
	for _, filter := range [][]string{{"service"}, {"front-end"}, {"front-end", "service"}} {
		for _, outputType := range []godot.OutputType{godot.OUT_PNG, godot.OUT_SVG} {
			generateGraph(filter, manifests, outputType, true)
			generateGraph(filter, manifests, outputType, false)
		}
	}
	log.Printf("Graphs generated in %v done.", outputPath)
}

func generateGraph(typeFilters []string, manifests Manifests, outputType godot.OutputType, requestFlow bool) {
	segments := []string{"graph"}
	segments = append(segments, typeFilters...)
	if requestFlow {
		segments = append(segments, "requests", "flow")
	} else {
		segments = append(segments, "dependencies")
	}
	fileName := strings.Join(segments, "-") + "." + string(outputType)
	dotter, err := godot.NewDotter(outputType, godot.GRAPH_DIRECTED, fileName)
	defer dotter.Close()

	if err != nil {
		log.Fatalf("Failed to access graphviz, make sure it's installed. error: %v", err)
	}

	manifestsRegistered := make(map[string]bool)

	for _, manifest := range manifests {
		if !manifest.Active {
			continue
		}
		var match = false
		for _, t := range manifest.Types {
			for _, f := range typeFilters {
				if t == f {
					match = true
				}
			}
		}
		if !match {
			continue
		}
		label := manifest.Name
		if len(manifest.Types) > 0 {
			label = manifest.Name + "\n(" + strings.Join(manifest.Types, " | ") + ")"
		}
		nodeName := strings.Replace(manifest.Name, " ", "-", -1)
		dotter.SetLabel(nodeName, label)
		manifestsRegistered[nodeName] = true
		for _, dependency := range manifest.Dependencies {
			if dependency.Implicit {
				continue
			}
			label := dependency.Name
			dependencyName := dependency.Name
			if dependency.Dedicated {
				dependencyName += "-" + manifest.Name
			}
			if dependency.UniqueName != "" {
				dependencyName = dependency.UniqueName
			}
			if dependency.External {
				dependencyName += "-external"
				label = dependency.Name + "\n(ext)"
			}
			dependencyName = strings.Replace(dependencyName, " ", "-", -1)
			// don't overwrite label name set by the main manifest
			if !manifestsRegistered[dependencyName] {
				dotter.SetLabel(dependencyName, label)
			}
			dotter.SetNodeShape(nodeName, godot.SHAPE_BOX)
			if requestFlow {
				if dependency.Direction == "" || dependency.Direction == "out" || dependency.Direction == "both" {
					dotter.SetLink(nodeName, dependencyName)
				}
				if dependency.Direction == "in" || dependency.Direction == "both" {
					dotter.SetLink(dependencyName, nodeName)
				}
			} else {
				dotter.SetLink(nodeName, dependencyName)
			}
		}
	}
}
