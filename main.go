/*
   mintop
   Copyright (c) 2021 Joshua Thompson-Lindley. All rights reserved.
   Licensed under the MIT License. See LICENSE file in the project root for full license information.
*/

package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/distatus/battery"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

// Formats bytes given by VirtualMemory into a gigabyte float64 value.
func formatBytesAsGB(value uint64) float64 {
	flValue := float64(value)
	flValue = flValue / float64(1000000000)
	flValue = math.Round(flValue*10) / 10

	return flValue
}

// Shorthand helper function to format a float64 as a string.
func ftos(value float64) string {
	return fmt.Sprint(value)
}

// Shorthand helper function to format an int as a string.
func itos(value int) string {
	return fmt.Sprint(value)
}

// Return the battery percentage and state. Assuming that 99.9% laptops have
// one battery, and will only display one battery status.
// This part won't appear if there are no batteries/not a laptop.
func getBatteryUsage() string {
	// Get all batteries
	batteries, batteryError := battery.GetAll()
	batteryCharge := 0.0
	batteryState := ""
	batteryErrors, partialError := batteryError.(battery.Errors)

	// If there are no battries, or if a fatal error occurred when reading the proc file,
	// then we won't display this section
	if _, isFatal := batteryError.(battery.ErrFatal); !isFatal && len(batteries) != 0 {
		for i, battery := range batteries {
			// Partial errors are fine, we can skip over that battery. Typically a partial error
			// occurrs when picking up a spare battery bay that is empty.
			if partialError && batteryErrors[i] != nil {
				continue
			}

			// Get the battery state and the charge by rounding the current capacity / full capacity of the battery
			// as a percentage.
			batteryState = battery.State.String()
			batteryCharge = math.Round(battery.Current / battery.Full * 100)
		}

		// Return the formatted section as a string.
		return "Battery Left: " + ftos(batteryCharge) + "% (" + batteryState + ")\n"
	}

	return ""
}

// Return the average CPU load and the amount of CPU cores, this includes hyperthreaded cores
// or whatever the AMD equivalent is called.
func getCpuUsage() string {
	// Get CPU percentage per core from gopsutil
	cpuCoreUsage, err := cpu.Percent(0, true)
	cpuUsage := 0.0
	cpuCores := 0

	if err == nil {
		// Iterate over core loads and total to work out the average.
		for i := range cpuCoreUsage {
			cpuUsage += cpuCoreUsage[i]
		}

		// While gopsutil can return the average load anyway, this provides an easy way to get all the cores.
		cpuCores = len(cpuCoreUsage)
		cpuUsage = math.Round(cpuUsage / float64(cpuCores))
	}

	// Return the formatted section as a string.
	return "CPU Usage: " + ftos(cpuUsage) + "% (" + itos(cpuCores) + " cores)\n"
}

// Return the memory usage as a percentage, as well the GB amounts of used memory.
func getMemoryUsage() string {
	// Get the memory information from gopsutil.
	memory, _ := mem.VirtualMemory()
	// Format the memory byte amounts given from the kernel as nice GB values to 1dp.
	usedMemory := formatBytesAsGB(memory.Used)
	totalMemory := formatBytesAsGB(memory.Total)
	// Round the percentage of used memory (eg. 50% instead of 50.000152%)
	memoryUsage := math.Round(memory.UsedPercent)

	// Return the formatted section as a string.
	return "Memory Usage: " + ftos(memoryUsage) + "% (" + ftos(usedMemory) + "GB/" + ftos(totalMemory) + "GB)\n"
}

// Draws UI elements every second.
func draw() {
	// Get all the formatted strings.
	cpu := getCpuUsage()
	ram := getMemoryUsage()
	batt := getBatteryUsage()
	// Get the terminal dimensions and create a new text widget.
	termWidth, termHeight := ui.TerminalDimensions()
	viewStats := widgets.NewParagraph()

	// Fill the terminal with the text widget and append all the strings together.
	viewStats.SetRect(0, 0, termWidth, termHeight)
	viewStats.Border = false
	viewStats.Text = cpu + ram + batt

	// Render the widget.
	ui.Render(viewStats)
}

// Entry point.
func main() {
	// Quit if an error occurs when initializing termui.
	if uiError := ui.Init(); uiError != nil {
		log.Fatalf("failed to initialize termui: %v", uiError)
	}

	defer ui.Close()

	// Begin the UI event loop, and start a second-long tick cycle.
	uiEvents := ui.PollEvents()
	tick := time.NewTicker(time.Second).C

	// Draw first rather than wait a second for the UI to render.
	draw()

	// Iterate over the UI events and tick channel.
	for {
		select {
		case event := <-uiEvents:
			// Quit the application if the [q] or [Crtl-C] command was given.
			switch event.ID {
			case "q", "<C-c>":
				return
			}
		// Redraw the UI every tick.
		case <-tick:
			draw()
		}
	}
}
