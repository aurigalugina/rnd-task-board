import { describe, expect, it } from "vitest";
import { validateCwd } from "./cwdValidation.js";

const HOME = "/home/dev";

describe("validateCwd", () => {
  it("menolak relative path", () => {
    expect(validateCwd("projects/foo", HOME)).toEqual({
      ok: false,
      reason: "cwd harus berupa absolute path",
    });
  });

  it("menolak root filesystem", () => {
    const result = validateCwd("/", HOME);
    expect(result.ok).toBe(false);
  });

  it("menolak home directory itu sendiri", () => {
    const result = validateCwd("/home/dev", HOME);
    expect(result.ok).toBe(false);
  });

  it("menolak home directory dengan trailing slash", () => {
    const result = validateCwd("/home/dev/", HOME);
    expect(result.ok).toBe(false);
  });

  it("menerima folder project di dalam home", () => {
    expect(validateCwd("/home/dev/projects/rnd-task-board", HOME)).toEqual({ ok: true });
  });

  it("menerima folder project di luar home (mis. /srv/repos)", () => {
    expect(validateCwd("/srv/repos/foo", HOME)).toEqual({ ok: true });
  });

  it("menolak extra forbidden root yang dikonfigurasi", () => {
    const result = validateCwd("/data", HOME, ["/data"]);
    expect(result.ok).toBe(false);
  });
});
