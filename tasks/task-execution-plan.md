# Poros Task Execution Plan

**Generated:** 2025-12-18
**PRD Version:** 1.0

## Execution Phases

### Phase 1: MVP Foundation (v0.1.0)
**Goal:** Working ICMP traceroute on Linux with basic output
**Duration:** 2-3 weeks
**Tasks:** T001-T025 (25 tasks)

#### Week 1: Project Setup & Core Infrastructure
| Day | Tasks | Focus |
|-----|-------|-------|
| 1-2 | T001, T006 | Go module, dependencies |
| 3-4 | T002, T003 | Core data structures, interfaces |
| 5 | T004, T005 | Build system, CLI framework |

#### Week 2: ICMP Probe Implementation
| Day | Tasks | Focus |
|-----|-------|-------|
| 1 | T007 | ICMP checksum |
| 2-3 | T008, T010 | Raw socket, TTL |
| 4-5 | T009, T011 | Packet builder, send/receive |

#### Week 3: Tracer & Output
| Day | Tasks | Focus |
|-----|-------|-------|
| 1-2 | T012, T013 | Response parser, tests |
| 3 | T014, T015 | Tracer core, DNS |
| 4 | T016, T017, T018 | Sequential trace |
| 5 | T020, T021, T024, T025 | Basic output |

**Deliverable:** `sudo poros google.com` works on Linux

---

### Phase 2: Core Features (v0.2.0)
**Goal:** UDP probe, concurrent mode, cross-platform basics
**Duration:** 2-3 weeks
**Tasks:** T026-T038 (13 tasks)

#### Week 4: UDP Probe
| Day | Tasks | Focus |
|-----|-------|-------|
| 1 | T026 | UDP packet builder |
| 2-3 | T027, T028 | UDP socket, send/receive |
| 4-5 | T029, T030, T031 | Parser, integration |

#### Week 5: Concurrent Tracing
| Day | Tasks | Focus |
|-----|-------|-------|
| 1-2 | T032, T033 | Pool manager, collector |
| 3-4 | T034, T035 | Concurrent logic, rate limiting |
| 5 | T036, T037, T038 | Early termination, CLI, tests |

**Deliverable:** Fast concurrent traces, UDP probe option

---

### Phase 3: Enrichment (v0.3.0)
**Goal:** ASN, GeoIP, rDNS enrichment with caching
**Duration:** 2 weeks
**Tasks:** T039-T053 (15 tasks)

#### Week 6: Lookup Implementations
| Day | Tasks | Focus |
|-----|-------|-------|
| 1 | T039 | rDNS lookup |
| 2 | T040 | LRU cache |
| 3 | T041, T042 | MaxMind, ASN |
| 4-5 | T043, T044 | GeoIP, orchestrator |

#### Week 7: Integration & Output Formats
| Day | Tasks | Focus |
|-----|-------|-------|
| 1-2 | T045, T046 | Tracer integration, CLI |
| 3 | T047, T048 | DB download, tests |
| 4-5 | T049, T050, T051, T052, T053 | JSON/CSV output |

**Deliverable:** Enriched traces with multiple output formats

---

### Phase 4: Advanced Probes (v0.4.0)
**Goal:** TCP SYN, Paris mode, Windows/macOS support
**Duration:** 3 weeks
**Tasks:** T054-T075 (22 tasks)

#### Week 8: TCP SYN Probe
| Day | Tasks | Focus |
|-----|-------|-------|
| 1 | T054, T056 | TCP packet, IP header |
| 2-3 | T055, T057 | Raw socket, send/receive |
| 4-5 | T058, T059, T060 | Cleanup, integration |

#### Week 9: Paris Traceroute
| Day | Tasks | Focus |
|-----|-------|-------|
| 1-2 | T061, T062 | Flow hash, ICMP |
| 3-4 | T063, T064 | UDP, multipath |
| 5 | T065, T066 | CLI, docs |

#### Week 10: Platform Support
| Day | Tasks | Focus |
|-----|-------|-------|
| 1-2 | T067, T068 | macOS sockets |
| 3-4 | T069, T070 | Windows sockets |
| 5 | T071, T072, T073, T074, T075 | Errors, interfaces, builds |

**Deliverable:** Professional network analysis tool

---

### Phase 5: TUI & Polish (v0.5.0)
**Goal:** Interactive TUI, HTML reports, documentation
**Duration:** 2-3 weeks
**Tasks:** T076-T089 (14 tasks)

#### Week 11: TUI Framework
| Day | Tasks | Focus |
|-----|-------|-------|
| 1 | T076 | Bubble Tea setup |
| 2-3 | T077, T078 | Update, view |
| 4-5 | T079, T080 | Real-time, continuous |

#### Week 12: TUI Features & HTML
| Day | Tasks | Focus |
|-----|-------|-------|
| 1-2 | T081, T082, T083, T084 | Export, CLI, themes, tests |
| 3-4 | T085, T086, T087 | HTML template, table, chart |
| 5 | T088, T089 | HTML CLI, tests |

**Deliverable:** Full-featured TUI and export options

---

### Phase 6: Release (v1.0.0)
**Goal:** Production-ready release with packaging
**Duration:** 1-2 weeks
**Tasks:** T090-T097 (8 tasks)

#### Week 13-14: Release Preparation
| Day | Tasks | Focus |
|-----|-------|-------|
| 1-2 | T090, T091 | GoReleaser, CI |
| 3 | T092, T093, T094 | Package managers, Docker |
| 4 | T095, T096 | README, man page |
| 5 | T097 | Final testing |

**Deliverable:** v1.0.0 release

---

## Critical Path

The following tasks are on the critical path and must be completed in sequence:

```
T001 → T002 → T003 → T008 → T011 → T012 → T014 → T016 → T018 → T025
  │                                                              │
  └──────────────── MVP COMPLETE ────────────────────────────────┘

T025 → T032 → T034 → T037
  │                    │
  └── CONCURRENT ──────┘

T025 → T039 → T044 → T045
  │                    │
  └── ENRICHMENT ──────┘

T025 → T076 → T079 → T082
  │                    │
  └──── TUI ───────────┘
```

## Parallel Execution Opportunities

These task groups can be worked on in parallel:

### During Phase 1 (MVP)
- T020-T023 (Output formatters) in parallel with T016-T018 (Tracer)
- T004 (Makefile) can be done anytime

### During Phase 2 (Core)
- T026-T031 (UDP) in parallel with T032-T038 (Concurrent)

### During Phase 3 (Enrichment)
- T049-T053 (JSON/CSV) in parallel with T039-T048 (Enrichment)

### During Phase 4 (Advanced)
- T054-T060 (TCP) in parallel with T061-T066 (Paris)
- T067-T075 (Platform) can overlap with probe development

### During Phase 5 (TUI)
- T076-T084 (TUI) in parallel with T085-T089 (HTML)

## Risk Mitigation

### High-Risk Tasks
| Task | Risk | Mitigation |
|------|------|------------|
| T008 | Raw socket permissions | Document CAP_NET_RAW early |
| T069 | Windows socket API | Research early, have fallback |
| T067 | macOS BPF restrictions | Test on actual hardware |
| T057 | TCP connection handling | Ensure proper RST cleanup |

### Recommended Buffer Time
- Add 1 week buffer after Phase 4 (platform support is complex)
- Add 2-3 days buffer before release

## Resource Requirements

### Development Environment
- Linux (primary development)
- macOS (for testing)
- Windows (for testing)
- VMs or cloud instances for all platforms

### External Dependencies
- MaxMind GeoLite2 databases (free registration)
- GitHub account for releases
- Homebrew tap repository

## Success Metrics

| Metric | Target | Measured At |
|--------|--------|-------------|
| Test coverage | >70% | Each phase |
| Binary size | <15MB | Phase 6 |
| 30-hop trace time | <5s | Phase 2 |
| Memory usage | <50MB | Phase 2 |
| Documentation | Complete | Phase 6 |

## Notes

- Start each day by updating task status
- Commit after completing each task
- Run full test suite before marking phase complete
- Keep PRD updated if requirements change
- Document any deviations from plan
