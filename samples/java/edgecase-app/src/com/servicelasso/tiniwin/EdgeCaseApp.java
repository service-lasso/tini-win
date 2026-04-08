package com.servicelasso.tiniwin;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Duration;
import java.util.HashMap;
import java.util.List;
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
            case "spawn-child-shell" -> runSpawnChildShell(options);
            case "runtime-exec-array" -> runRuntimeExecArray(options);
            case "runtime-exec-string" -> runRuntimeExecString(options);
            case "batch-wrapper-child" -> runBatchWrapperChild(options);
            case "relaunch-orphan" -> runRelaunchOrphan(options);
            case "broker" -> runBroker(options);
            case "client" -> runClient(options);
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
        Process child = startAndRecord(options, List.of("cmd", "/c", pingCommand(options)));
        System.out.println("java-edgecase: ProcessBuilder direct child pid=" + child.pid());
        waitForDuration(options);
    }

    private static void runSpawnChildShell(Map<String, String> options) throws Exception {
        Process child = startAndRecord(options, List.of("cmd", "/c", pingCommand(options)));
        System.out.println("java-edgecase: ProcessBuilder shell child pid=" + child.pid());
        waitForDuration(options);
    }

    private static void runRuntimeExecArray(Map<String, String> options) throws Exception {
        Process child = Runtime.getRuntime().exec(new String[]{"cmd", "/c", pingCommand(options)});
        recordChild(options, child.pid());
        System.out.println("java-edgecase: Runtime.exec array child pid=" + child.pid());
        waitForDuration(options);
    }

    private static void runRuntimeExecString(Map<String, String> options) throws Exception {
        Process child = Runtime.getRuntime().exec("cmd /c " + pingCommand(options));
        recordChild(options, child.pid());
        System.out.println("java-edgecase: Runtime.exec string child pid=" + child.pid());
        waitForDuration(options);
    }

    private static void runBatchWrapperChild(Map<String, String> options) throws Exception {
        Path helper = resolveBatchHelper();
        int duration = parseInt(options.getOrDefault("duration", "30"), 30);
        Process child = new ProcessBuilder("cmd", "/c", helper.toString(), Integer.toString(duration)).start();
        recordChild(options, child.pid());
        System.out.println("java-edgecase: batch-wrapper child pid=" + child.pid());
        waitForDuration(options);
    }

    private static void runRelaunchOrphan(Map<String, String> options) throws Exception {
        Process child = new ProcessBuilder("cmd", "/c", pingCommand(options)).start();
        recordChild(options, child.pid());
        System.out.println("java-edgecase: relaunch-orphan spawned pid=" + child.pid() + " and exiting parent immediately");
    }

    private static void runBroker(Map<String, String> options) throws Exception {
        String requestFile = required(options, "request-file", "--request-file is required for broker mode");
        String stopFile = required(options, "stop-file", "--stop-file is required for broker mode");
        int duration = parseInt(options.getOrDefault("duration", "30"), 30);
        Path requestPath = Path.of(requestFile);
        Path stopPath = Path.of(stopFile);
        writePid(options.get("pid-file"), ProcessHandle.current().pid());
        Files.deleteIfExists(requestPath);
        Files.deleteIfExists(stopPath);
        System.out.println("java-edgecase: broker ready");
        while (true) {
            if (Files.exists(stopPath)) {
                System.out.println("java-edgecase: broker stopping");
                return;
            }
            if (Files.exists(requestPath)) {
                Files.deleteIfExists(requestPath);
                Process child = new ProcessBuilder("cmd", "/c", "ping 127.0.0.1 -n " + duration + " >nul").start();
                recordChild(options, child.pid());
                System.out.println("java-edgecase: broker spawned pid=" + child.pid());
            }
            Thread.sleep(200);
        }
    }

    private static void runClient(Map<String, String> options) throws Exception {
        String requestFile = required(options, "request-file", "--request-file is required for client mode");
        Files.writeString(Path.of(requestFile), "spawn", StandardCharsets.UTF_8);
        System.out.println("java-edgecase: client requested broker spawn");
        while (true) {
            Thread.sleep(1000);
        }
    }

    private static Process startAndRecord(Map<String, String> options, List<String> command) throws IOException {
        Process child = new ProcessBuilder(command).start();
        recordChild(options, child.pid());
        return child;
    }

    private static void recordChild(Map<String, String> options, long pid) throws IOException {
        writePid(options.get("child-pid-file"), pid);
    }

    private static void waitForDuration(Map<String, String> options) throws InterruptedException {
        int duration = parseInt(options.getOrDefault("duration", "30"), 30);
        Thread.sleep(Duration.ofSeconds(duration).toMillis());
    }

    private static String pingCommand(Map<String, String> options) {
        int duration = parseInt(options.getOrDefault("duration", "30"), 30);
        return "ping 127.0.0.1 -n " + duration + " >nul";
    }

    private static Path resolveBatchHelper() {
        String classPath = System.getProperty("java.class.path", "");
        String firstEntry = classPath.split(java.io.File.pathSeparator)[0];
        Path firstPath = Path.of(firstEntry).toAbsolutePath();
        Path baseDir = Files.isDirectory(firstPath) ? firstPath : firstPath.getParent();
        if (baseDir == null) {
            fail("unable to resolve java batch helper base directory", 2);
        }
        Path helper = baseDir.resolve("child-wrapper.cmd");
        if (!Files.exists(helper)) {
            fail("batch helper not found: " + helper, 2);
        }
        return helper;
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
