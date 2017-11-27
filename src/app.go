package main

import (
    "io"
    "net/http"
    "encoding/xml"
    "os/exec"
    "log"
    "os"
    "regexp"
)

const LISTEN_ADDRESS = ":9202"
const NVIDIA_SMI_PATH = "/usr/bin/nvidia-smi"

var testMode string;

type NvidiaSmiLog struct {
    DriverVersion string `xml:"driver_version"`
    AttachedGPUs string `xml:"attached_gpus"`
    GPUs []struct {
        ProductName string `xml:"product_name"`
        ProductBrand string `xml:"product_brand"`
        UUID string `xml:"uuid"`
        MinorNumber string `xml:"minor_number"`
        FanSpeed string `xml:"fan_speed"`
        PCI struct {
            PCIBus string `xml:"pci_bus"`
        } `xml:"pci"`
        FbMemoryUsage struct {
            Total string `xml:"total"`
            Used string `xml:"used"`
            Free string `xml:"free"`
        } `xml:"fb_memory_usage"`
        Utilization struct {
            GPUUtil string `xml:"gpu_util"`
            MemoryUtil string `xml:"memory_util"`
        } `xml:"utilization"`
        Temperature struct {
            GPUTemp string `xml:"gpu_temp"`
            GPUTempMaxThreshold string `xml:"gpu_temp_max_threshold"`
            GPUTempSlowThreshold string `xml:"gpu_temp_slow_threshold"`
        } `xml:"temperature"`
        PowerReadings struct {
            PowerDraw string `xml:"power_draw"`
            PowerLimit string `xml:"power_limit"`
        } `xml:"power_readings"`
        Clocks struct {
            GraphicsClock string `xml:"graphics_clock"`
            SmClock string `xml:"sm_clock"`
            MemClock string `xml:"mem_clock"`
            VideoClock string `xml:"video_clock"`
        } `xml:"clocks"`
        MaxClocks struct {
            GraphicsClock string `xml:"graphics_clock"`
            SmClock string `xml:"sm_clock"`
            MemClock string `xml:"mem_clock"`
            VideoClock string `xml:"video_clock"`
        } `xml:"max_clocks"`
    } `xml:"gpu"`
}

func formatValue(key string, meta string, value string) string {
    result := key;
    if (meta != "") {
        result += "{" + meta + "}";
    }
    return result + " " + value +"\n"
}

func filterNumber(value string) string {
    r := regexp.MustCompile("[^0-9.]")
    return r.ReplaceAllString(value, "")
}

func writeMetric(w http.ResponseWriter, name string, meta string, value string) (n int, err error) {
  filtered_value := filterNumber(value)

  if filtered_value != "" {
    return io.WriteString(w, formatValue(name, meta, filtered_value))
  } else {
    return 0, nil
  }
}

func metrics(w http.ResponseWriter, r *http.Request) {
    log.Print("Serving /metrics")

    var cmd *exec.Cmd
    if (testMode == "1") {
        dir, err := os.Getwd()
        if err != nil {
            log.Fatal(err)
        }
        cmd = exec.Command("/bin/cat", dir + "/test.xml")
    } else {
        cmd = exec.Command(NVIDIA_SMI_PATH, "-q", "-x")
    }

    // Execute system command
    stdout, err := cmd.Output()
    if err != nil {
        println(err.Error())
        return
    }

    // Parse XML
    var xmlData NvidiaSmiLog
    xml.Unmarshal(stdout, &xmlData)

    // Output
    io.WriteString(w, formatValue("nvidiasmi_driver_version", "", xmlData.DriverVersion))
    io.WriteString(w, formatValue("nvidiasmi_attached_gpus", "", filterNumber(xmlData.AttachedGPUs)))
    for _, GPU := range xmlData.GPUs {
        meta := "name=\"" + GPU.ProductName + " [" + GPU.MinorNumber + "]\"" + ", " + "uuid=\"" + GPU.UUID + "\""

        writeMetric(w, "nvidiasmi_fan_speed", meta, GPU.FanSpeed)
        writeMetric(w, "nvidiasmi_memory_usage_total", meta, GPU.FbMemoryUsage.Total)
        writeMetric(w, "nvidiasmi_memory_usage_used", meta, GPU.FbMemoryUsage.Used)
        writeMetric(w, "nvidiasmi_memory_usage_free", meta, GPU.FbMemoryUsage.Free)
        writeMetric(w, "nvidiasmi_utilization_gpu", meta, GPU.Utilization.GPUUtil)
        writeMetric(w, "nvidiasmi_utilization_memory", meta, GPU.Utilization.MemoryUtil)
        writeMetric(w, "nvidiasmi_temp_gpu", meta, GPU.Temperature.GPUTemp)
        writeMetric(w, "nvidiasmi_temp_gpu_max", meta, GPU.Temperature.GPUTempMaxThreshold)
        writeMetric(w, "nvidiasmi_temp_gpu_slow", meta, GPU.Temperature.GPUTempSlowThreshold)
        writeMetric(w, "nvidiasmi_power_draw", meta, GPU.PowerReadings.PowerDraw)
        writeMetric(w, "nvidiasmi_power_limit", meta, GPU.PowerReadings.PowerLimit)
        writeMetric(w, "nvidiasmi_clock_graphics", meta, GPU.Clocks.GraphicsClock)
        writeMetric(w, "nvidiasmi_clock_graphics_max", meta, GPU.MaxClocks.GraphicsClock)
        writeMetric(w, "nvidiasmi_clock_sm", meta, GPU.Clocks.SmClock)
        writeMetric(w, "nvidiasmi_clock_sm_max", meta, GPU.MaxClocks.SmClock)
        writeMetric(w, "nvidiasmi_clock_mem", meta, GPU.Clocks.MemClock)
        writeMetric(w, "nvidiasmi_clock_mem_max", meta, GPU.MaxClocks.MemClock)
        writeMetric(w, "nvidiasmi_clock_video", meta, GPU.Clocks.VideoClock)
        writeMetric(w, "nvidiasmi_clock_video_max", meta, GPU.MaxClocks.VideoClock)
    }
}

func index(w http.ResponseWriter, r *http.Request) {
    log.Print("Serving /index")
    html := `<!doctype html>
<html>
    <head>
        <meta charset="utf-8">
        <title>Nvidia SMI Exporter</title>
    </head>
    <body>
        <h1>Nvidia SMI Exporter</h1>
        <p><a href="/metrics">Metrics</a></p>
    </body>
</html>`
    io.WriteString(w, html)
}

func main() {
    testMode = os.Getenv("TEST_MODE")
    if (testMode == "1") {
        log.Print("Test mode is enabled")
    }

    log.Print("Nvidia SMI exporter listening on " + LISTEN_ADDRESS)
    http.HandleFunc("/", index)
    http.HandleFunc("/metrics", metrics)
    http.ListenAndServe(LISTEN_ADDRESS, nil)
}
