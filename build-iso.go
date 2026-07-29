package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
    args := os.Args[1:]
    if len(args) < 2 {
        printUsage()
    }

    sourceISO := args[0]
    ksFile := args[1]
    outputISO := "custom.iso"
    if len(args) >= 3 {
        outputISO = args[2]
    }

    // ---------- Step 1: check / install required tools ----------
    log.Println("Checking for required tools (mkksiso, ksvalidator)...")

    var missing []string
    if !commandExists("mkksiso"){
        missing = append(missing, "lorax")
    }

    if !commandExists("ksvalidator"){
        missing = append(missing, "pykickstart")
    }

    if len(missing) > 0 {
        log.Printf("Missing packages: %v. Installing...\n", missing)
        installArgs := append([]string{"dnf", "install", "-y"}, missing...)
        run ("sudo", installArgs...)
    } else {
        log.Println("mkksiso and ksvalidator already installed")
    }

    // ---------- Step 2: validate source ISO ----------
    log.Println("Checking source ISO...")
    if !fileExists(sourceISO) {
        log.Fatal("Source ISO not found: " + sourceISO + "\n")
    }
    log.Printf("Found source ISO: %s (%s)", sourceISO, sizeHumanReadable(sourceISO))

    // ---------- Step 3: validate kickstart file ----------
    log.Println("Checking kickstart file...")
    if !fileExists(ksFile) {
        log.Fatal("Kickstart file not found: " + ksFile + "\n")
    }
    log.Println("Found kickstart file")

    log.Println("Validating kickstart syntax...")
    run("ksvalidator", ksFile)
    log.Println("Kickstart file is valid")

    // ---------- Step 4: build the ISO ----------
    log.Println("Building automated ISO (this may take some time)...")
    run("mkksiso", "--ks", ksFile, sourceISO, outputISO)

    // ---------- Step 5: confirm output ----------
    if fileExists(outputISO) {
        log.Println("Build succeeded!")
        log.Printf("Output ISO: %s (%s)\n", outputISO, sizeHumanReadable(outputISO))
    } else {
        log.Fatalln("mkksiso reported success but output ISO was not found.")
    }

}

func printUsage() {
    log.Fatalln("Usage: build-iso <source.iso> <ks.cfg> [output.iso]")
}

func commandExists(cmd string) bool {
    _, err := exec.LookPath(cmd)
    return err == nil
}

func run(name string, args ...string) {
    cmd := exec.Command(name, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        log.Fatalf("command failed: %s %v (%v)\n", name, args, err)
    }
}

func fileExists(path string) bool {
    info, err := os.Stat(path)
    return err == nil && !info.IsDir()
}

func sizeHumanReadable(path string) string {
    info, err := os.Stat(path)
    if err != nil {
        return "unknown size"
    }
    bytes := float64(info.Size())
    units := []string{"B", "KB", "MB", "GB"}
    i := 0

    for bytes >= 1024 && i < len(units) - 1 {
        bytes /= 1024
        i++
    }

    return fmt.Sprintf("%.1f%s", bytes, units[i])
}