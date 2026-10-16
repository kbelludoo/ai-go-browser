# Browser task comparison

Generated: 2026-10-11T18:59:05Z. Mode: explicit public URLs (read-only smoke checks). Fixture version: local-v1.

Chromium: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

Baseline navigation: NavigateReady: loader-matched lifecycle load event, without a fixed settling delay.

Environment: windows/amd64, go1.27.0, 28 logical CPUs. 1 repetitions per engine.

| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ai_go | 81/100 | 257640.73 | 0.00 | 24849 | 58.05 | 94.26 | 2441.42 |
| chromium_cdp | 87/100 | 315054.81 | 361.45 | 20120 | 3.35 | 16.68 | 3031.38 |

## Individual tasks

Cold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.

| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |
|---|---:|---|---|---|---:|---:|---:|---:|---:|
| ai_go | 1 | public_1 | cold | true | 1146.45 | 0.00 | 1146.45 | 29293 | 253162 |
| ai_go | 1 | public_2 | warm | true | 2631.98 | 0.00 | 2631.98 | 33849 | 2017633 |
| ai_go | 1 | public_3 | warm | true | 2313.05 | 0.00 | 2313.05 | 35457 | 1879868 |
| ai_go | 1 | public_4 | warm | true | 2146.62 | 0.00 | 2146.62 | 37329 | 2879566 |
| ai_go | 1 | public_5 | warm | true | 2114.49 | 0.00 | 2114.49 | 36263 | 2074801 |
| ai_go | 1 | public_6 | warm | true | 2503.45 | 0.00 | 2503.45 | 34924 | 2113804 |
| ai_go | 1 | public_7 | warm | true | 2433.65 | 0.00 | 2433.65 | 38034 | 1867684 |
| ai_go | 1 | public_8 | warm | true | 2836.76 | 0.00 | 2836.76 | 41138 | 3250656 |
| ai_go | 1 | public_9 | warm | true | 1866.09 | 0.00 | 1866.09 | 34305 | 864264 |
| ai_go | 1 | public_10 | warm | true | 2540.12 | 0.00 | 2540.12 | 33604 | 1282156 |
| ai_go | 1 | public_11 | warm | true | 1509.47 | 0.00 | 1509.47 | 32374 | 255130 |
| ai_go | 1 | public_12 | warm | true | 1099.04 | 0.00 | 1099.04 | 27788 | 101785 |
| ai_go | 1 | public_13 | warm | true | 1169.54 | 0.00 | 1169.54 | 28822 | 170799 |
| ai_go | 1 | public_14 | warm | true | 1425.36 | 0.00 | 1425.36 | 37109 | 127179 |
| ai_go | 1 | public_15 | warm | true | 1302.36 | 0.00 | 1302.36 | 28390 | 154145 |
| ai_go | 1 | public_16 | warm | true | 1186.98 | 0.00 | 1186.98 | 31375 | 132193 |
| ai_go | 1 | public_17 | warm | true | 971.39 | 0.00 | 971.39 | 28196 | 95576 |
| ai_go | 1 | public_18 | warm | true | 1233.82 | 0.00 | 1233.82 | 28594 | 134375 |
| ai_go | 1 | public_19 | warm | true | 1205.18 | 0.00 | 1205.18 | 28086 | 89690 |
| ai_go | 1 | public_20 | warm | true | 1600.20 | 0.00 | 1600.20 | 30696 | 213142 |
| ai_go | 1 | public_21 | warm | true | 645.75 | 0.00 | 645.75 | 20425 | 64185 |
| ai_go | 1 | public_22 | warm | true | 245.60 | 0.00 | 245.60 | 11811 | 34115 |
| ai_go | 1 | public_23 | warm | true | 995.49 | 0.00 | 995.49 | 21057 | 64185 |
| ai_go | 1 | public_24 | warm | true | 560.01 | 0.00 | 560.01 | 24620 | 34212 |
| ai_go | 1 | public_25 | warm | true | 182.42 | 0.00 | 182.42 | 28972 | 41430 |
| ai_go | 1 | public_26 | warm | false | 20000.51 | 0.00 | 20000.51 | 317 | 5664 |
| ai_go | 1 | public_27 | warm | false | 20000.73 | 0.00 | 20000.73 | 313 | 5660 |
| ai_go | 1 | public_28 | warm | false | 20000.03 | 0.00 | 20000.03 | 315 | 5662 |
| ai_go | 1 | public_29 | warm | false | 20000.08 | 0.00 | 20000.08 | 313 | 5660 |
| ai_go | 1 | public_30 | warm | true | 1952.68 | 0.00 | 1952.68 | 28604 | 554379 |
| ai_go | 1 | public_31 | warm | true | 770.86 | 0.00 | 770.86 | 27285 | 305073 |
| ai_go | 1 | public_32 | warm | true | 260.21 | 0.00 | 260.21 | 14226 | 19625 |
| ai_go | 1 | public_33 | warm | true | 515.84 | 0.00 | 515.84 | 6915 | 19158 |
| ai_go | 1 | public_34 | warm | true | 398.61 | 0.00 | 398.61 | 9550 | 18594 |
| ai_go | 1 | public_35 | warm | true | 242.69 | 0.00 | 242.69 | 32345 | 222343 |
| ai_go | 1 | public_36 | warm | true | 263.90 | 0.00 | 263.90 | 24932 | 121671 |
| ai_go | 1 | public_37 | warm | true | 1075.81 | 0.00 | 1075.81 | 9882 | 19192 |
| ai_go | 1 | public_38 | warm | true | 593.29 | 0.00 | 593.29 | 19678 | 132696 |
| ai_go | 1 | public_39 | warm | false | 20000.27 | 0.00 | 20000.27 | 310 | 5657 |
| ai_go | 1 | public_40 | warm | true | 397.63 | 0.00 | 397.63 | 34229 | 372902 |
| ai_go | 1 | public_41 | warm | true | 61.66 | 0.00 | 61.66 | 11937 | 258828 |
| ai_go | 1 | public_42 | warm | true | 159.92 | 0.00 | 159.92 | 13755 | 531762 |
| ai_go | 1 | public_43 | warm | true | 2827.88 | 0.00 | 2827.88 | 26627 | 287968 |
| ai_go | 1 | public_44 | warm | true | 914.39 | 0.00 | 914.39 | 17128 | 59963 |
| ai_go | 1 | public_45 | warm | true | 243.88 | 0.00 | 243.88 | 11297 | 12607 |
| ai_go | 1 | public_46 | warm | true | 48.09 | 0.00 | 48.09 | 15530 | 41608 |
| ai_go | 1 | public_47 | warm | true | 692.64 | 0.00 | 692.64 | 21407 | 500829 |
| ai_go | 1 | public_48 | warm | true | 935.78 | 0.00 | 935.78 | 10775 | 185401 |
| ai_go | 1 | public_49 | warm | true | 797.60 | 0.00 | 797.60 | 44384 | 208259 |
| ai_go | 1 | public_50 | warm | true | 102.89 | 0.00 | 102.89 | 8332 | 15863 |
| ai_go | 1 | public_51 | warm | true | 55.70 | 0.00 | 55.70 | 10864 | 19148 |
| ai_go | 1 | public_52 | warm | true | 4153.59 | 0.00 | 4153.59 | 17417 | 31584 |
| ai_go | 1 | public_53 | warm | true | 171.41 | 0.00 | 171.41 | 23669 | 217469 |
| ai_go | 1 | public_54 | warm | true | 73.22 | 0.00 | 73.22 | 21476 | 51924 |
| ai_go | 1 | public_55 | warm | true | 849.94 | 0.00 | 849.94 | 27669 | 54040 |
| ai_go | 1 | public_56 | warm | true | 227.36 | 0.00 | 227.36 | 339 | 1145 |
| ai_go | 1 | public_57 | warm | true | 88.62 | 0.00 | 88.62 | 14484 | 192265 |
| ai_go | 1 | public_58 | warm | true | 1381.46 | 0.00 | 1381.46 | 1906 | 6688 |
| ai_go | 1 | public_59 | warm | true | 205.96 | 0.00 | 205.96 | 1701 | 2394 |
| ai_go | 1 | public_60 | warm | true | 706.71 | 0.00 | 706.71 | 1078 | 755 |
| ai_go | 1 | public_61 | warm | true | 269.37 | 0.00 | 269.37 | 24849 | 44341 |
| ai_go | 1 | public_62 | warm | true | 456.00 | 0.00 | 456.00 | 28412 | 110756 |
| ai_go | 1 | public_63 | warm | true | 1084.56 | 0.00 | 1084.56 | 46205 | 432718 |
| ai_go | 1 | public_64 | warm | true | 805.93 | 0.00 | 805.93 | 8835 | 185164 |
| ai_go | 1 | public_65 | warm | true | 2057.61 | 0.00 | 2057.61 | 4952 | 19304 |
| ai_go | 1 | public_66 | warm | true | 383.69 | 0.00 | 383.69 | 11000 | 83108 |
| ai_go | 1 | public_67 | warm | true | 3487.41 | 0.00 | 3487.41 | 12409 | 52219 |
| ai_go | 1 | public_68 | warm | true | 2257.43 | 0.00 | 2257.43 | 13593 | 52978 |
| ai_go | 1 | public_69 | warm | true | 3027.60 | 0.00 | 3027.60 | 22670 | 368531 |
| ai_go | 1 | public_70 | warm | true | 5114.56 | 0.00 | 5114.56 | 24972 | 148429 |
| ai_go | 1 | public_71 | warm | true | 4652.72 | 0.00 | 4652.72 | 15691 | 344194 |
| ai_go | 1 | public_72 | warm | false | 8633.50 | 0.00 | 8633.50 | 353 | 120 |
| ai_go | 1 | public_73 | warm | false | 20000.05 | 0.00 | 20000.05 | 0 | 0 |
| ai_go | 1 | public_74 | warm | true | 1328.95 | 0.00 | 1328.95 | 2739 | 75594 |
| ai_go | 1 | public_75 | warm | true | 1476.80 | 0.00 | 1476.80 | 5312 | 8026 |
| ai_go | 1 | public_76 | warm | true | 1749.17 | 0.00 | 1749.17 | 21716 | 100312 |
| ai_go | 1 | public_77 | warm | true | 180.15 | 0.00 | 180.15 | 5440 | 6658 |
| ai_go | 1 | public_78 | warm | true | 451.59 | 0.00 | 451.59 | 26386 | 335610 |
| ai_go | 1 | public_79 | warm | true | 5930.66 | 0.00 | 5930.66 | 60486 | 1510989 |
| ai_go | 1 | public_80 | warm | true | 5722.58 | 0.00 | 5722.58 | 47352 | 540803 |
| ai_go | 1 | public_81 | warm | true | 1914.11 | 0.00 | 1914.11 | 40294 | 243994 |
| ai_go | 1 | public_82 | warm | true | 1697.56 | 0.00 | 1697.56 | 31677 | 102718 |
| ai_go | 1 | public_83 | warm | true | 2985.85 | 0.00 | 2985.85 | 72164 | 1087945 |
| ai_go | 1 | public_84 | warm | true | 1913.79 | 0.00 | 1913.79 | 28001 | 516590 |
| ai_go | 1 | public_85 | warm | true | 6398.07 | 0.00 | 6398.07 | 42733 | 1000702 |
| ai_go | 1 | public_86 | warm | true | 1591.59 | 0.00 | 1591.59 | 39918 | 319632 |
| ai_go | 1 | public_87 | warm | true | 5468.69 | 0.00 | 5468.69 | 62508 | 639794 |
| ai_go | 1 | public_88 | warm | true | 1185.12 | 0.00 | 1185.12 | 58653 | 1468128 |
| ai_go | 1 | public_89 | warm | false | 6377.38 | 0.00 | 6377.38 | 0 | 0 |
| ai_go | 1 | public_90 | warm | false | 0.00 | 0.00 | 0.00 | 0 | 0 |
| ai_go | 1 | public_91 | warm | false | 0.00 | 0.00 | 0.00 | 0 | 0 |
| ai_go | 1 | public_92 | warm | false | 0.00 | 0.00 | 0.00 | 0 | 0 |
| ai_go | 1 | public_93 | warm | false | 0.00 | 0.00 | 0.00 | 0 | 0 |
| ai_go | 1 | public_94 | warm | false | 0.00 | 0.00 | 0.00 | 0 | 0 |
| ai_go | 1 | public_95 | warm | false | 0.00 | 0.00 | 0.00 | 0 | 0 |
| ai_go | 1 | public_96 | warm | false | 0.06 | 0.00 | 0.06 | 0 | 0 |
| ai_go | 1 | public_97 | warm | false | 0.00 | 0.00 | 0.00 | 0 | 0 |
| ai_go | 1 | public_98 | warm | false | 0.00 | 0.00 | 0.00 | 0 | 0 |
| ai_go | 1 | public_99 | warm | false | 0.58 | 0.00 | 0.58 | 0 | 0 |
| ai_go | 1 | public_100 | warm | false | 0.52 | 0.00 | 0.52 | 0 | 0 |
| chromium_cdp | 1 | public_1 | cold | true | 2078.92 | 361.45 | 2440.37 | 23474 | 315315 |
| chromium_cdp | 1 | public_2 | warm | true | 7603.83 | 0.00 | 7603.83 | 32900 | 2398474 |
| chromium_cdp | 1 | public_3 | warm | false | 11074.27 | 0.00 | 11074.27 | 0 | 0 |
| chromium_cdp | 1 | public_4 | warm | true | 1713.91 | 0.00 | 1713.91 | 37954 | 3183600 |
| chromium_cdp | 1 | public_5 | warm | true | 2110.00 | 0.00 | 2110.00 | 36142 | 2416587 |
| chromium_cdp | 1 | public_6 | warm | true | 1981.03 | 0.00 | 1981.03 | 35080 | 2630100 |
| chromium_cdp | 1 | public_7 | warm | true | 3218.91 | 0.00 | 3218.91 | 57372 | 2208406 |
| chromium_cdp | 1 | public_8 | warm | true | 1718.79 | 0.00 | 1718.79 | 58522 | 3489278 |
| chromium_cdp | 1 | public_9 | warm | true | 2100.38 | 0.00 | 2100.38 | 32635 | 1060513 |
| chromium_cdp | 1 | public_10 | warm | true | 2056.25 | 0.00 | 2056.25 | 33808 | 1534826 |
| chromium_cdp | 1 | public_11 | warm | true | 8434.10 | 0.00 | 8434.10 | 24993 | 255196 |
| chromium_cdp | 1 | public_12 | warm | true | 1973.35 | 0.00 | 1973.35 | 20120 | 103159 |
| chromium_cdp | 1 | public_13 | warm | true | 2493.77 | 0.00 | 2493.77 | 26134 | 237006 |
| chromium_cdp | 1 | public_14 | warm | true | 2461.23 | 0.00 | 2461.23 | 31075 | 607280 |
| chromium_cdp | 1 | public_15 | warm | true | 2279.90 | 0.00 | 2279.90 | 23128 | 210068 |
| chromium_cdp | 1 | public_16 | warm | true | 2671.96 | 0.00 | 2671.96 | 26005 | 317311 |
| chromium_cdp | 1 | public_17 | warm | true | 2172.49 | 0.00 | 2172.49 | 23190 | 96995 |
| chromium_cdp | 1 | public_18 | warm | true | 3801.11 | 0.00 | 3801.11 | 22783 | 431868 |
| chromium_cdp | 1 | public_19 | warm | true | 2704.61 | 0.00 | 2704.61 | 25439 | 147523 |
| chromium_cdp | 1 | public_20 | warm | true | 2560.13 | 0.00 | 2560.13 | 23767 | 273462 |
| chromium_cdp | 1 | public_21 | warm | true | 2396.10 | 0.00 | 2396.10 | 17008 | 62705 |
| chromium_cdp | 1 | public_22 | warm | true | 1130.24 | 0.00 | 1130.24 | 10807 | 35843 |
| chromium_cdp | 1 | public_23 | warm | true | 2806.76 | 0.00 | 2806.76 | 17646 | 62641 |
| chromium_cdp | 1 | public_24 | warm | true | 939.38 | 0.00 | 939.38 | 22022 | 34331 |
| chromium_cdp | 1 | public_25 | warm | true | 233.71 | 0.00 | 233.71 | 25835 | 42403 |
| chromium_cdp | 1 | public_26 | warm | false | 245.35 | 0.00 | 245.35 | 773 | 28681 |
| chromium_cdp | 1 | public_27 | warm | false | 231.77 | 0.00 | 231.77 | 757 | 28644 |
| chromium_cdp | 1 | public_28 | warm | false | 242.85 | 0.00 | 242.85 | 765 | 28652 |
| chromium_cdp | 1 | public_29 | warm | false | 227.39 | 0.00 | 227.39 | 757 | 28644 |
| chromium_cdp | 1 | public_30 | warm | true | 2206.85 | 0.00 | 2206.85 | 17251 | 555874 |
| chromium_cdp | 1 | public_31 | warm | true | 1266.26 | 0.00 | 1266.26 | 18247 | 305511 |
| chromium_cdp | 1 | public_32 | warm | false | 11192.04 | 0.00 | 11192.04 | 0 | 0 |
| chromium_cdp | 1 | public_33 | warm | true | 490.40 | 0.00 | 490.40 | 6855 | 18361 |
| chromium_cdp | 1 | public_34 | warm | true | 271.97 | 0.00 | 271.97 | 7364 | 18417 |
| chromium_cdp | 1 | public_35 | warm | true | 328.50 | 0.00 | 328.50 | 27811 | 139963 |
| chromium_cdp | 1 | public_36 | warm | true | 365.70 | 0.00 | 365.70 | 22159 | 75727 |
| chromium_cdp | 1 | public_37 | warm | true | 2806.53 | 0.00 | 2806.53 | 8406 | 20742 |
| chromium_cdp | 1 | public_38 | warm | true | 6280.06 | 0.00 | 6280.06 | 23536 | 132510 |
| chromium_cdp | 1 | public_39 | warm | false | 228.04 | 0.00 | 228.04 | 745 | 28632 |
| chromium_cdp | 1 | public_40 | warm | true | 2046.93 | 0.00 | 2046.93 | 39750 | 595662 |
| chromium_cdp | 1 | public_41 | warm | true | 375.32 | 0.00 | 375.32 | 9812 | 258913 |
| chromium_cdp | 1 | public_42 | warm | true | 530.67 | 0.00 | 530.67 | 8991 | 531464 |
| chromium_cdp | 1 | public_43 | warm | true | 3695.57 | 0.00 | 3695.57 | 22841 | 313987 |
| chromium_cdp | 1 | public_44 | warm | true | 5518.66 | 0.00 | 5518.66 | 14656 | 65877 |
| chromium_cdp | 1 | public_45 | warm | true | 2158.47 | 0.00 | 2158.47 | 9891 | 12448 |
| chromium_cdp | 1 | public_46 | warm | true | 2508.22 | 0.00 | 2508.22 | 13351 | 61610 |
| chromium_cdp | 1 | public_47 | warm | true | 5414.02 | 0.00 | 5414.02 | 18722 | 694687 |
| chromium_cdp | 1 | public_48 | warm | true | 2030.25 | 0.00 | 2030.25 | 9743 | 403748 |
| chromium_cdp | 1 | public_49 | warm | true | 2931.77 | 0.00 | 2931.77 | 30125 | 328648 |
| chromium_cdp | 1 | public_50 | warm | true | 407.67 | 0.00 | 407.67 | 6143 | 15516 |
| chromium_cdp | 1 | public_51 | warm | true | 185.13 | 0.00 | 185.13 | 9523 | 19042 |
| chromium_cdp | 1 | public_52 | warm | false | 20000.16 | 0.00 | 20000.16 | 0 | 0 |
| chromium_cdp | 1 | public_53 | warm | false | 1302.17 | 0.00 | 1302.17 | 263 | 8620 |
| chromium_cdp | 1 | public_54 | warm | true | 1234.46 | 0.00 | 1234.46 | 19868 | 53537 |
| chromium_cdp | 1 | public_55 | warm | true | 2406.71 | 0.00 | 2406.71 | 24060 | 52993 |
| chromium_cdp | 1 | public_56 | warm | false | 20000.89 | 0.00 | 20000.89 | 0 | 0 |
| chromium_cdp | 1 | public_57 | warm | true | 915.91 | 0.00 | 915.91 | 12096 | 193151 |
| chromium_cdp | 1 | public_58 | warm | true | 819.26 | 0.00 | 819.26 | 1443 | 6242 |
| chromium_cdp | 1 | public_59 | warm | true | 118.00 | 0.00 | 118.00 | 2851 | 2358 |
| chromium_cdp | 1 | public_60 | warm | true | 659.63 | 0.00 | 659.63 | 926 | 646 |
| chromium_cdp | 1 | public_61 | warm | true | 207.13 | 0.00 | 207.13 | 21392 | 37786 |
| chromium_cdp | 1 | public_62 | warm | true | 116.06 | 0.00 | 116.06 | 21941 | 94684 |
| chromium_cdp | 1 | public_63 | warm | true | 601.02 | 0.00 | 601.02 | 29317 | 397301 |
| chromium_cdp | 1 | public_64 | warm | true | 648.19 | 0.00 | 648.19 | 7093 | 182527 |
| chromium_cdp | 1 | public_65 | warm | true | 1702.58 | 0.00 | 1702.58 | 4334 | 17828 |
| chromium_cdp | 1 | public_66 | warm | true | 239.88 | 0.00 | 239.88 | 28738 | 77343 |
| chromium_cdp | 1 | public_67 | warm | true | 3275.28 | 0.00 | 3275.28 | 9564 | 49790 |
| chromium_cdp | 1 | public_68 | warm | true | 1408.11 | 0.00 | 1408.11 | 11987 | 44001 |
| chromium_cdp | 1 | public_69 | warm | true | 3334.60 | 0.00 | 3334.60 | 34181 | 358173 |
| chromium_cdp | 1 | public_70 | warm | true | 4924.58 | 0.00 | 4924.58 | 26422 | 131870 |
| chromium_cdp | 1 | public_71 | warm | true | 4967.11 | 0.00 | 4967.11 | 11351 | 337263 |
| chromium_cdp | 1 | public_72 | warm | false | 5471.40 | 0.00 | 5471.40 | 237 | 105 |
| chromium_cdp | 1 | public_73 | warm | false | 20000.65 | 0.00 | 20000.65 | 0 | 0 |
| chromium_cdp | 1 | public_74 | warm | false | 20000.49 | 0.00 | 20000.49 | 228 | 8569 |
| chromium_cdp | 1 | public_75 | warm | true | 389.06 | 0.00 | 389.06 | 4314 | 6926 |
| chromium_cdp | 1 | public_76 | warm | true | 1610.61 | 0.00 | 1610.61 | 18285 | 80354 |
| chromium_cdp | 1 | public_77 | warm | true | 1282.01 | 0.00 | 1282.01 | 4995 | 6036 |
| chromium_cdp | 1 | public_78 | warm | true | 194.76 | 0.00 | 194.76 | 24213 | 333265 |
| chromium_cdp | 1 | public_79 | warm | true | 3374.06 | 0.00 | 3374.06 | 35466 | 1489738 |
| chromium_cdp | 1 | public_80 | warm | true | 3425.32 | 0.00 | 3425.32 | 37476 | 530086 |
| chromium_cdp | 1 | public_81 | warm | true | 1102.54 | 0.00 | 1102.54 | 35379 | 235468 |
| chromium_cdp | 1 | public_82 | warm | true | 1150.40 | 0.00 | 1150.40 | 28970 | 98081 |
| chromium_cdp | 1 | public_83 | warm | true | 2147.46 | 0.00 | 2147.46 | 59569 | 1159045 |
| chromium_cdp | 1 | public_84 | warm | true | 2126.63 | 0.00 | 2126.63 | 26428 | 509953 |
| chromium_cdp | 1 | public_85 | warm | true | 5065.19 | 0.00 | 5065.19 | 32961 | 957589 |
| chromium_cdp | 1 | public_86 | warm | true | 1688.41 | 0.00 | 1688.41 | 26636 | 311854 |
| chromium_cdp | 1 | public_87 | warm | true | 926.56 | 0.00 | 926.56 | 38303 | 751346 |
| chromium_cdp | 1 | public_88 | warm | true | 973.43 | 0.00 | 973.43 | 41525 | 1454383 |
| chromium_cdp | 1 | public_89 | warm | true | 7104.09 | 0.00 | 7104.09 | 44993 | 2549700 |
| chromium_cdp | 1 | public_90 | warm | true | 16676.67 | 0.00 | 16676.67 | 31891 | 1673859 |
| chromium_cdp | 1 | public_91 | warm | true | 2238.13 | 0.00 | 2238.13 | 852 | 9483 |
| chromium_cdp | 1 | public_92 | warm | true | 198.24 | 0.00 | 198.24 | 787 | 723 |
| chromium_cdp | 1 | public_93 | warm | true | 1688.94 | 0.00 | 1688.94 | 19961 | 148378 |
| chromium_cdp | 1 | public_94 | warm | true | 836.55 | 0.00 | 836.55 | 1287 | 29375 |
| chromium_cdp | 1 | public_95 | warm | true | 1051.68 | 0.00 | 1051.68 | 20830 | 263739 |
| chromium_cdp | 1 | public_96 | warm | true | 2752.84 | 0.00 | 2752.84 | 15356 | 24974 |
| chromium_cdp | 1 | public_97 | warm | true | 2078.09 | 0.00 | 2078.09 | 15221 | 47256 |
| chromium_cdp | 1 | public_98 | warm | true | 4183.86 | 0.00 | 4183.86 | 19545 | 1826400 |
| chromium_cdp | 1 | public_99 | warm | true | 1803.64 | 0.00 | 1803.64 | 12451 | 261388 |
| chromium_cdp | 1 | public_100 | warm | true | 5368.35 | 0.00 | 5368.35 | 9800 | 35365 |

## Errors

- ai_go, run 1, public_26: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 1, public_27: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 1, public_28: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 1, public_29: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 1, public_39: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 1, public_72: public smoke returned HTTP 504
- ai_go, run 1, public_73: open: live navigation: context deadline exceeded
- ai_go, run 1, public_89: open: live page connection closed: cdp event queue overflow (capacity 256)
- ai_go, run 1, public_90: open: cdp connection closed: cdp event queue overflow (capacity 256)
- ai_go, run 1, public_91: open: cdp connection closed: cdp event queue overflow (capacity 256)
- ai_go, run 1, public_92: open: cdp connection closed: cdp event queue overflow (capacity 256)
- ai_go, run 1, public_93: open: cdp connection closed: cdp event queue overflow (capacity 256)
- ai_go, run 1, public_94: open: cdp connection closed: cdp event queue overflow (capacity 256)
- ai_go, run 1, public_95: open: cdp connection closed: cdp event queue overflow (capacity 256)
- ai_go, run 1, public_96: open: cdp connection closed: cdp event queue overflow (capacity 256)
- ai_go, run 1, public_97: open: cdp connection closed: cdp event queue overflow (capacity 256)
- ai_go, run 1, public_98: open: cdp connection closed: cdp event queue overflow (capacity 256)
- ai_go, run 1, public_99: open: cdp connection closed: cdp event queue overflow (capacity 256)
- ai_go, run 1, public_100: open: cdp connection closed: cdp event queue overflow (capacity 256)
- chromium_cdp, run 1, public_3: navigate https://es.wikipedia.org/wiki/Espana: net::ERR_NAME_NOT_RESOLVED
- chromium_cdp, run 1, public_26: public smoke returned HTTP 403
- chromium_cdp, run 1, public_27: public smoke returned HTTP 403
- chromium_cdp, run 1, public_28: public smoke returned HTTP 403
- chromium_cdp, run 1, public_29: public smoke returned HTTP 403
- chromium_cdp, run 1, public_32: navigate https://docs.python.org/3/: net::ERR_NAME_NOT_RESOLVED
- chromium_cdp, run 1, public_39: public smoke returned HTTP 403
- chromium_cdp, run 1, public_52: context deadline exceeded
- chromium_cdp, run 1, public_53: public smoke returned HTTP 403
- chromium_cdp, run 1, public_56: navigation readiness: context deadline exceeded
- chromium_cdp, run 1, public_72: public smoke returned HTTP 504
- chromium_cdp, run 1, public_73: context deadline exceeded
- chromium_cdp, run 1, public_74: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded

## Interpretation limits

- This corpus cannot prove access to every website or universal superiority. No external AI model is run and token counts are not measured.
- Each engine and repetition runs in a fresh isolated Go worker. The first case includes engine setup; subsequent cases reuse that engine. The first JS case may still include a lazy Chromium launch for AI Go.
- The baseline is real Chromium controlled through the repository's CDP/stealth launcher and NavigateReady. It waits for the navigation's lifecycle load event without a fixed settling delay, then polls task assertions; it is not a comparison against every agent framework.
- AI Go uses CompactSnapshots (CLI default), omitting automatic Markdown. Compact snapshot bytes are actual UTF-8 JSON lengths. Both adapters cap visible text at 5000 Unicode code points, links at 200 and headings at 100; schemas differ. Document bytes represent fetched/serialized HTML, not model input tokens or total network traffic.
- GoHeapAlloc and GoSys belong to the benchmark worker and include fixtures/adapter allocations. GoSys is runtime-reserved memory, not RSS. Harness RSS and Chromium tree RSS are separate OS samples after each task, not continuously measured peaks.
- Chromium tree RSS sums resident working sets of process descendants. Shared pages can be counted multiple times, processes can exit during sampling, and unavailable readings remain null rather than zero.
- Per-case contexts bound CDP operations and assertion polling; AI Go commands also use the browser's internal request/render timeouts and may exceed the requested case timeout.
- Results depend on hardware, OS, Chromium version, local caches and task order. The command alternates engine order across repetitions; aggregate medians across different task types describe this fixed corpus only.
- Very short local HTTP tasks can round to zero at the host timer resolution; these readings do not mean that the operation has no cost.
- Public checks require a nonempty document and reject reported HTTP errors and a small set of obvious challenge/error phrases. They do not prove semantic completion or distinguish every challenge from usable content; no forms or account actions are performed.
