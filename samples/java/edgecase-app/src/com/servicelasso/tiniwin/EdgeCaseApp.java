package com.servicelasso.tiniwin;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Duration;
import java.util.HashMap;
import java.util.Map;

public final class EdgeCaseApp {
    public static void main(String[] args) throws Exception {
        Map<String, String> options = parseArgs(args);
        String mode = options.getOrDefault("mode", "simple-exit");
        writePid(options.get("pid-file"), ProcessHandle.current().pid());

        switch (mode) {
            case "simple-exit" -> runSimpleExit(options);
            case "graceful-stop" -> runGracefulStop(options);
            case "ignore-stop" -> runIgnoreStop();
            case "spawn-child" -> runSpawnChild(options);
            default -> fail("unknown mode: " + mode, 2);
        }
    }

    private static void runSimpleExit(Map<String, String> options) throws Exception {
        System.out.println("java-edgecase: simple-exit starting");
        int sleepMs = parseInt(options.getOrDefault("sleep-ms", "0"), 0);
        if (sleepMs > 0) {
            Thread.sleep(sleepMs);
        }
        int exitCode = parseInt(options.getOrDefault("exit-code", "0"), 0);
        System.out.println("java-edgecase: exiting " + exitCode);
        System.exit(exitCode);
    }

    private static void runGracefulStop(Map<String, String> options) throws Exception {
        String signalFile = required(options, "signal-file", "--signal-file is required for graceful-stop mode");
        if (Boolean.parseBoolean(options.getOrDefault("send", "false"))) {
            Files.writeString(Path.of(signalFile), "stop", StandardCharsets.UTF_8);
            System.out.println("java-edgecase: graceful signal sent");
            return;
        }
        System.out.println("java-edgecase: graceful-stop running");
        while (true) {
            if (Files.exists(Path.of(signalFile))) {
                System.out.println("java-edgecase: graceful-stop detected signal, exiting 0");
                return;
            }
            Thread.sleep(250);
        }
    }

    private static void runIgnoreStop() throws Exception {
        System.out.println("java-edgecase: ignore-stop running");
        while (true) {
            Thread.sleep(2000);
            System.out.println("java-edgecase: still alive");
        }
    }

    private static void runSpawnChild(Map<String, String> options) throws Exception {
        int duration = parseInt(options.getOrDefault("duration", "30"), 30);
        Process child = new ProcessBuilder("cmd", "/c", "ping 127.0.0.1 -n " + duration + " >nul").start();
        writePid(options.get("child-pid-file"), child.pid());
        System.out.println("java-edgecase: spawned pid=" + child.pid());
        Thread.sleep(Duration.ofSeconds(duration).toMillis());
    }

    private static void writePid(String path, long pid) throws IOException {
        if (path == null || path.isBlank()) {
            return;
        }
        Files.writeString(Path.of(path), Long.toString(pid), StandardCharsets.UTF_8);
    }

    private static Map<String, String> parseArgs(String[] args) {
        Map<String, String> options = new HashMap<>();
        for (int i = 0; i < args.length; i++) {
            String arg = args[i];
            if (!arg.startsWith("--")) {
                continue;
            }
            String key = arg.substring(2);
            String value = "true";
            if (i + 1 < args.length && !args[i + 1].startsWith("--")) {
                value = args[++i];
            }
            options.put(key, value);
        }
        return options;
    }

    private static String required(Map<String, String> options, String key, String message) {
        String value = options.get(key);
        if (value == null || value.isBlank()) {
            fail(message, 2);
        }
        return value;
    }

    private static int parseInt(String value, int fallback) {
        try {
            return Integer.parseInt(value);
        } catch (NumberFormatException ex) {
            return fallback;
        }
    }

    private static void fail(String message, int code) {
        System.err.println(message);
        System.exit(code);
    }
}
