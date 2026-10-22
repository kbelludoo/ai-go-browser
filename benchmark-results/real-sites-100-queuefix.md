# Browser task comparison

Generated: 2026-10-17T02:32:46Z. Mode: explicit public URLs (read-only smoke checks). Fixture version: local-v1.

Chromium: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`

Baseline navigation: NavigateReady: loader-matched lifecycle load event, without a fixed settling delay.

Environment: windows/amd64, go1.27.0, 28 logical CPUs. 5 repetitions per engine.

| Engine | Task assertions passed | Mean corpus time including startup (ms) | Mean startup (ms) | Median compact JSON bytes | Max sampled Go heap (MiB) | Max sampled harness RSS (MiB) | Max sampled Chromium tree RSS (MiB) |
|---|---:|---:|---:|---:|---:|---:|---:|
| ai_go | 452/500 | 341172.67 | 0.00 | 22169 | 58.23 | 94.68 | 2068.47 |
| chromium_cdp | 444/500 | 305230.42 | 378.05 | 19981 | 5.06 | 18.73 | 2242.88 |

## Individual tasks

Cold rows include startup in total time. Warm rows reuse the engine; AI Go may lazily start Chromium on its first JS case. Snapshot extraction and assertion polling are included in task time. Memory sampling runs afterwards.

| Engine | Run | Case | Phase | Success | Task ms | Startup ms | Total ms | Compact JSON bytes | Document bytes |
|---|---:|---|---|---|---:|---:|---:|---:|---:|
| ai_go | 1 | public_1 | cold | true | 1100.76 | 0.00 | 1100.76 | 29611 | 253943 |
| ai_go | 1 | public_2 | warm | true | 1652.91 | 0.00 | 1652.91 | 33849 | 2017444 |
| ai_go | 1 | public_3 | warm | true | 1669.55 | 0.00 | 1669.55 | 34992 | 1876166 |
| ai_go | 1 | public_4 | warm | true | 1830.89 | 0.00 | 1830.89 | 37329 | 2873942 |
| ai_go | 1 | public_5 | warm | true | 1694.17 | 0.00 | 1694.17 | 36305 | 2075994 |
| ai_go | 1 | public_6 | warm | true | 1663.43 | 0.00 | 1663.43 | 34924 | 2113993 |
| ai_go | 1 | public_7 | warm | true | 1609.38 | 0.00 | 1609.38 | 38034 | 1866457 |
| ai_go | 1 | public_8 | warm | true | 1860.06 | 0.00 | 1860.06 | 41138 | 3249113 |
| ai_go | 1 | public_9 | warm | true | 1384.02 | 0.00 | 1384.02 | 34305 | 864585 |
| ai_go | 1 | public_10 | warm | true | 1470.11 | 0.00 | 1470.11 | 33604 | 1282423 |
| ai_go | 1 | public_11 | warm | true | 1292.08 | 0.00 | 1292.08 | 32245 | 254179 |
| ai_go | 1 | public_12 | warm | true | 1350.19 | 0.00 | 1350.19 | 27573 | 102830 |
| ai_go | 1 | public_13 | warm | true | 1398.93 | 0.00 | 1398.93 | 28417 | 174324 |
| ai_go | 1 | public_14 | warm | true | 1551.43 | 0.00 | 1551.43 | 37086 | 127227 |
| ai_go | 1 | public_15 | warm | true | 1419.79 | 0.00 | 1419.79 | 27983 | 156046 |
| ai_go | 1 | public_16 | warm | true | 1535.77 | 0.00 | 1535.77 | 31502 | 132580 |
| ai_go | 1 | public_17 | warm | true | 1356.49 | 0.00 | 1356.49 | 28243 | 95737 |
| ai_go | 1 | public_18 | warm | true | 1529.23 | 0.00 | 1529.23 | 28722 | 135080 |
| ai_go | 1 | public_19 | warm | true | 992.05 | 0.00 | 992.05 | 28738 | 91459 |
| ai_go | 1 | public_20 | warm | true | 1300.56 | 0.00 | 1300.56 | 30696 | 213319 |
| ai_go | 1 | public_21 | warm | true | 669.50 | 0.00 | 669.50 | 20425 | 64188 |
| ai_go | 1 | public_22 | warm | true | 253.43 | 0.00 | 253.43 | 11811 | 34115 |
| ai_go | 1 | public_23 | warm | true | 1499.71 | 0.00 | 1499.71 | 21057 | 64188 |
| ai_go | 1 | public_24 | warm | true | 747.53 | 0.00 | 747.53 | 24595 | 34039 |
| ai_go | 1 | public_25 | warm | true | 197.29 | 0.00 | 197.29 | 28427 | 41147 |
| ai_go | 1 | public_26 | warm | false | 20000.75 | 0.00 | 20000.75 | 317 | 5653 |
| ai_go | 1 | public_27 | warm | false | 20000.94 | 0.00 | 20000.94 | 313 | 5649 |
| ai_go | 1 | public_28 | warm | false | 20000.27 | 0.00 | 20000.27 | 315 | 5651 |
| ai_go | 1 | public_29 | warm | false | 20000.30 | 0.00 | 20000.30 | 313 | 5649 |
| ai_go | 1 | public_30 | warm | true | 2180.76 | 0.00 | 2180.76 | 28864 | 608655 |
| ai_go | 1 | public_31 | warm | true | 652.82 | 0.00 | 652.82 | 27325 | 300785 |
| ai_go | 1 | public_32 | warm | true | 101.42 | 0.00 | 101.42 | 14226 | 19625 |
| ai_go | 1 | public_33 | warm | true | 276.96 | 0.00 | 276.96 | 7838 | 20123 |
| ai_go | 1 | public_34 | warm | true | 372.06 | 0.00 | 372.06 | 9550 | 18594 |
| ai_go | 1 | public_35 | warm | true | 564.55 | 0.00 | 564.55 | 32345 | 222341 |
| ai_go | 1 | public_36 | warm | true | 309.38 | 0.00 | 309.38 | 25000 | 121746 |
| ai_go | 1 | public_37 | warm | true | 1089.78 | 0.00 | 1089.78 | 9882 | 19190 |
| ai_go | 1 | public_38 | warm | true | 704.21 | 0.00 | 704.21 | 19678 | 132697 |
| ai_go | 1 | public_39 | warm | false | 20000.62 | 0.00 | 20000.62 | 310 | 5646 |
| ai_go | 1 | public_40 | warm | true | 94.23 | 0.00 | 94.23 | 34229 | 372981 |
| ai_go | 1 | public_41 | warm | true | 89.31 | 0.00 | 89.31 | 11937 | 258828 |
| ai_go | 1 | public_42 | warm | true | 218.02 | 0.00 | 218.02 | 13755 | 531745 |
| ai_go | 1 | public_43 | warm | true | 3232.82 | 0.00 | 3232.82 | 26627 | 286124 |
| ai_go | 1 | public_44 | warm | true | 1323.09 | 0.00 | 1323.09 | 17076 | 59911 |
| ai_go | 1 | public_45 | warm | true | 611.37 | 0.00 | 611.37 | 11297 | 12617 |
| ai_go | 1 | public_46 | warm | true | 307.33 | 0.00 | 307.33 | 15530 | 41608 |
| ai_go | 1 | public_47 | warm | true | 110.36 | 0.00 | 110.36 | 21407 | 500979 |
| ai_go | 1 | public_48 | warm | true | 987.42 | 0.00 | 987.42 | 10775 | 185401 |
| ai_go | 1 | public_49 | warm | true | 658.60 | 0.00 | 658.60 | 44520 | 209845 |
| ai_go | 1 | public_50 | warm | true | 117.50 | 0.00 | 117.50 | 8333 | 15864 |
| ai_go | 1 | public_51 | warm | true | 503.84 | 0.00 | 503.84 | 10866 | 19150 |
| ai_go | 1 | public_52 | warm | true | 525.69 | 0.00 | 525.69 | 17166 | 31411 |
| ai_go | 1 | public_53 | warm | false | 20000.57 | 0.00 | 20000.57 | 306 | 8293 |
| ai_go | 1 | public_54 | warm | true | 388.56 | 0.00 | 388.56 | 22068 | 52491 |
| ai_go | 1 | public_55 | warm | true | 1039.31 | 0.00 | 1039.31 | 27669 | 53921 |
| ai_go | 1 | public_56 | warm | true | 399.19 | 0.00 | 399.19 | 339 | 1145 |
| ai_go | 1 | public_57 | warm | true | 423.09 | 0.00 | 423.09 | 14484 | 192265 |
| ai_go | 1 | public_58 | warm | true | 2438.77 | 0.00 | 2438.77 | 1906 | 6688 |
| ai_go | 1 | public_59 | warm | true | 223.65 | 0.00 | 223.65 | 1701 | 2394 |
| ai_go | 1 | public_60 | warm | true | 1256.22 | 0.00 | 1256.22 | 1078 | 755 |
| ai_go | 1 | public_61 | warm | true | 299.92 | 0.00 | 299.92 | 24849 | 44341 |
| ai_go | 1 | public_62 | warm | true | 986.94 | 0.00 | 986.94 | 28325 | 117674 |
| ai_go | 1 | public_63 | warm | true | 1070.72 | 0.00 | 1070.72 | 46205 | 432718 |
| ai_go | 1 | public_64 | warm | true | 685.13 | 0.00 | 685.13 | 8857 | 183808 |
| ai_go | 1 | public_65 | warm | true | 1990.44 | 0.00 | 1990.44 | 4952 | 19304 |
| ai_go | 1 | public_66 | warm | true | 402.86 | 0.00 | 402.86 | 11000 | 83835 |
| ai_go | 1 | public_67 | warm | true | 3568.32 | 0.00 | 3568.32 | 12300 | 52338 |
| ai_go | 1 | public_68 | warm | true | 1491.06 | 0.00 | 1491.06 | 13505 | 53362 |
| ai_go | 1 | public_69 | warm | false | 20000.41 | 0.00 | 20000.41 | 0 | 0 |
| ai_go | 1 | public_70 | warm | true | 3606.77 | 0.00 | 3606.77 | 24477 | 147230 |
| ai_go | 1 | public_71 | warm | true | 4486.67 | 0.00 | 4486.67 | 15893 | 344943 |
| ai_go | 1 | public_72 | warm | true | 1292.40 | 0.00 | 1292.40 | 13324 | 33081 |
| ai_go | 1 | public_73 | warm | false | 20000.11 | 0.00 | 20000.11 | 0 | 0 |
| ai_go | 1 | public_74 | warm | true | 1369.89 | 0.00 | 1369.89 | 2739 | 77046 |
| ai_go | 1 | public_75 | warm | true | 1541.45 | 0.00 | 1541.45 | 5396 | 8237 |
| ai_go | 1 | public_76 | warm | true | 2184.38 | 0.00 | 2184.38 | 21716 | 100312 |
| ai_go | 1 | public_77 | warm | true | 1944.70 | 0.00 | 1944.70 | 5259 | 6559 |
| ai_go | 1 | public_78 | warm | true | 500.55 | 0.00 | 500.55 | 26708 | 335708 |
| ai_go | 1 | public_79 | warm | true | 5668.25 | 0.00 | 5668.25 | 60481 | 1492038 |
| ai_go | 1 | public_80 | warm | true | 3646.59 | 0.00 | 3646.59 | 46226 | 521015 |
| ai_go | 1 | public_81 | warm | true | 2153.07 | 0.00 | 2153.07 | 35696 | 223069 |
| ai_go | 1 | public_82 | warm | true | 1099.52 | 0.00 | 1099.52 | 34687 | 115627 |
| ai_go | 1 | public_83 | warm | true | 3434.96 | 0.00 | 3434.96 | 69729 | 1009600 |
| ai_go | 1 | public_84 | warm | true | 1948.50 | 0.00 | 1948.50 | 28988 | 560145 |
| ai_go | 1 | public_85 | warm | true | 5657.24 | 0.00 | 5657.24 | 42784 | 1105204 |
| ai_go | 1 | public_86 | warm | true | 1173.63 | 0.00 | 1173.63 | 42732 | 335244 |
| ai_go | 1 | public_87 | warm | true | 4552.51 | 0.00 | 4552.51 | 60288 | 603153 |
| ai_go | 1 | public_88 | warm | true | 3648.93 | 0.00 | 3648.93 | 58209 | 1464269 |
| ai_go | 1 | public_89 | warm | false | 371.81 | 0.00 | 371.81 | 926 | 28509 |
| ai_go | 1 | public_90 | warm | true | 17499.04 | 0.00 | 17499.04 | 36993 | 1703965 |
| ai_go | 1 | public_91 | warm | true | 2221.78 | 0.00 | 2221.78 | 937 | 9603 |
| ai_go | 1 | public_92 | warm | true | 288.44 | 0.00 | 288.44 | 905 | 738 |
| ai_go | 1 | public_93 | warm | true | 2485.54 | 0.00 | 2485.54 | 22238 | 153178 |
| ai_go | 1 | public_94 | warm | true | 1031.73 | 0.00 | 1031.73 | 3043 | 31237 |
| ai_go | 1 | public_95 | warm | true | 1405.33 | 0.00 | 1405.33 | 22780 | 276107 |
| ai_go | 1 | public_96 | warm | true | 2793.94 | 0.00 | 2793.94 | 17366 | 28321 |
| ai_go | 1 | public_97 | warm | true | 1893.65 | 0.00 | 1893.65 | 14196 | 51470 |
| ai_go | 1 | public_98 | warm | true | 2924.53 | 0.00 | 2924.53 | 16336 | 1829957 |
| ai_go | 1 | public_99 | warm | true | 1851.36 | 0.00 | 1851.36 | 13549 | 265300 |
| ai_go | 1 | public_100 | warm | true | 6774.41 | 0.00 | 6774.41 | 11024 | 37267 |
| chromium_cdp | 1 | public_1 | cold | true | 2451.63 | 355.84 | 2807.47 | 23816 | 257486 |
| chromium_cdp | 1 | public_2 | warm | true | 2117.02 | 0.00 | 2117.02 | 32900 | 2398477 |
| chromium_cdp | 1 | public_3 | warm | true | 1755.56 | 0.00 | 1755.56 | 34966 | 2014471 |
| chromium_cdp | 1 | public_4 | warm | true | 2225.33 | 0.00 | 2225.33 | 39013 | 3451096 |
| chromium_cdp | 1 | public_5 | warm | true | 1304.44 | 0.00 | 1304.44 | 34683 | 2217883 |
| chromium_cdp | 1 | public_6 | warm | true | 2137.80 | 0.00 | 2137.80 | 35075 | 2630531 |
| chromium_cdp | 1 | public_7 | warm | true | 1951.85 | 0.00 | 1951.85 | 55777 | 2098210 |
| chromium_cdp | 1 | public_8 | warm | true | 2372.90 | 0.00 | 2372.90 | 58494 | 3487732 |
| chromium_cdp | 1 | public_9 | warm | true | 1885.89 | 0.00 | 1885.89 | 31344 | 963891 |
| chromium_cdp | 1 | public_10 | warm | true | 1375.42 | 0.00 | 1375.42 | 32071 | 1362175 |
| chromium_cdp | 1 | public_11 | warm | true | 2448.75 | 0.00 | 2448.75 | 24942 | 328680 |
| chromium_cdp | 1 | public_12 | warm | true | 2306.22 | 0.00 | 2306.22 | 19981 | 104735 |
| chromium_cdp | 1 | public_13 | warm | true | 2857.93 | 0.00 | 2857.93 | 25375 | 240727 |
| chromium_cdp | 1 | public_14 | warm | true | 3007.72 | 0.00 | 3007.72 | 31029 | 607328 |
| chromium_cdp | 1 | public_15 | warm | true | 2722.35 | 0.00 | 2722.35 | 24302 | 241316 |
| chromium_cdp | 1 | public_16 | warm | true | 2589.02 | 0.00 | 2589.02 | 25279 | 134921 |
| chromium_cdp | 1 | public_17 | warm | true | 2696.06 | 0.00 | 2696.06 | 23265 | 97158 |
| chromium_cdp | 1 | public_18 | warm | true | 4061.99 | 0.00 | 4061.99 | 22903 | 432573 |
| chromium_cdp | 1 | public_19 | warm | true | 2482.67 | 0.00 | 2482.67 | 25704 | 149456 |
| chromium_cdp | 1 | public_20 | warm | true | 2566.73 | 0.00 | 2566.73 | 28964 | 678266 |
| chromium_cdp | 1 | public_21 | warm | true | 3719.70 | 0.00 | 3719.70 | 17008 | 62708 |
| chromium_cdp | 1 | public_22 | warm | true | 689.15 | 0.00 | 689.15 | 10761 | 35843 |
| chromium_cdp | 1 | public_23 | warm | true | 3301.39 | 0.00 | 3301.39 | 17645 | 62663 |
| chromium_cdp | 1 | public_24 | warm | true | 1007.16 | 0.00 | 1007.16 | 21869 | 34093 |
| chromium_cdp | 1 | public_25 | warm | true | 266.34 | 0.00 | 266.34 | 24736 | 42197 |
| chromium_cdp | 1 | public_26 | warm | false | 407.53 | 0.00 | 407.53 | 773 | 28497 |
| chromium_cdp | 1 | public_27 | warm | false | 430.95 | 0.00 | 430.95 | 757 | 28481 |
| chromium_cdp | 1 | public_28 | warm | false | 436.07 | 0.00 | 436.07 | 765 | 28489 |
| chromium_cdp | 1 | public_29 | warm | false | 270.57 | 0.00 | 270.57 | 757 | 28481 |
| chromium_cdp | 1 | public_30 | warm | true | 2658.65 | 0.00 | 2658.65 | 19643 | 611022 |
| chromium_cdp | 1 | public_31 | warm | true | 1354.52 | 0.00 | 1354.52 | 18255 | 301465 |
| chromium_cdp | 1 | public_32 | warm | true | 414.04 | 0.00 | 414.04 | 12112 | 24203 |
| chromium_cdp | 1 | public_33 | warm | true | 286.53 | 0.00 | 286.53 | 6764 | 18268 |
| chromium_cdp | 1 | public_34 | warm | true | 373.34 | 0.00 | 373.34 | 7364 | 18417 |
| chromium_cdp | 1 | public_35 | warm | true | 357.18 | 0.00 | 357.18 | 27811 | 139963 |
| chromium_cdp | 1 | public_36 | warm | true | 389.50 | 0.00 | 389.50 | 22227 | 75796 |
| chromium_cdp | 1 | public_37 | warm | true | 2883.70 | 0.00 | 2883.70 | 8406 | 20740 |
| chromium_cdp | 1 | public_38 | warm | true | 6110.60 | 0.00 | 6110.60 | 23536 | 132512 |
| chromium_cdp | 1 | public_39 | warm | false | 261.87 | 0.00 | 261.87 | 745 | 28469 |
| chromium_cdp | 1 | public_40 | warm | true | 2386.71 | 0.00 | 2386.71 | 39823 | 595741 |
| chromium_cdp | 1 | public_41 | warm | true | 533.16 | 0.00 | 533.16 | 9812 | 258787 |
| chromium_cdp | 1 | public_42 | warm | true | 729.64 | 0.00 | 729.64 | 8991 | 531376 |
| chromium_cdp | 1 | public_43 | warm | true | 3897.85 | 0.00 | 3897.85 | 22718 | 310423 |
| chromium_cdp | 1 | public_44 | warm | true | 4740.08 | 0.00 | 4740.08 | 14381 | 64904 |
| chromium_cdp | 1 | public_45 | warm | true | 1776.73 | 0.00 | 1776.73 | 9891 | 12458 |
| chromium_cdp | 1 | public_46 | warm | true | 1818.35 | 0.00 | 1818.35 | 13351 | 61610 |
| chromium_cdp | 1 | public_47 | warm | true | 5043.53 | 0.00 | 5043.53 | 18722 | 694712 |
| chromium_cdp | 1 | public_48 | warm | true | 1849.90 | 0.00 | 1849.90 | 9664 | 400332 |
| chromium_cdp | 1 | public_49 | warm | true | 2774.42 | 0.00 | 2774.42 | 30254 | 330099 |
| chromium_cdp | 1 | public_50 | warm | true | 586.76 | 0.00 | 586.76 | 6144 | 15517 |
| chromium_cdp | 1 | public_51 | warm | true | 445.92 | 0.00 | 445.92 | 9525 | 19044 |
| chromium_cdp | 1 | public_52 | warm | false | 20000.24 | 0.00 | 20000.24 | 0 | 0 |
| chromium_cdp | 1 | public_53 | warm | false | 1748.85 | 0.00 | 1748.85 | 263 | 8620 |
| chromium_cdp | 1 | public_54 | warm | true | 1238.41 | 0.00 | 1238.41 | 19871 | 53531 |
| chromium_cdp | 1 | public_55 | warm | true | 2501.53 | 0.00 | 2501.53 | 24060 | 52874 |
| chromium_cdp | 1 | public_56 | warm | false | 20000.40 | 0.00 | 20000.40 | 0 | 0 |
| chromium_cdp | 1 | public_57 | warm | true | 763.66 | 0.00 | 763.66 | 12096 | 193151 |
| chromium_cdp | 1 | public_58 | warm | true | 848.48 | 0.00 | 848.48 | 1443 | 6242 |
| chromium_cdp | 1 | public_59 | warm | true | 143.87 | 0.00 | 143.87 | 2851 | 2358 |
| chromium_cdp | 1 | public_60 | warm | true | 736.43 | 0.00 | 736.43 | 926 | 646 |
| chromium_cdp | 1 | public_61 | warm | true | 202.11 | 0.00 | 202.11 | 21392 | 37786 |
| chromium_cdp | 1 | public_62 | warm | true | 120.45 | 0.00 | 120.45 | 21833 | 100363 |
| chromium_cdp | 1 | public_63 | warm | true | 614.75 | 0.00 | 614.75 | 29317 | 397301 |
| chromium_cdp | 1 | public_64 | warm | true | 612.13 | 0.00 | 612.13 | 7122 | 181523 |
| chromium_cdp | 1 | public_65 | warm | true | 1672.08 | 0.00 | 1672.08 | 4334 | 17828 |
| chromium_cdp | 1 | public_66 | warm | true | 277.50 | 0.00 | 277.50 | 28883 | 77867 |
| chromium_cdp | 1 | public_67 | warm | true | 3229.06 | 0.00 | 3229.06 | 9455 | 49905 |
| chromium_cdp | 1 | public_68 | warm | true | 1404.79 | 0.00 | 1404.79 | 11873 | 50416 |
| chromium_cdp | 1 | public_69 | warm | true | 18562.77 | 0.00 | 18562.77 | 33712 | 347636 |
| chromium_cdp | 1 | public_70 | warm | true | 3397.90 | 0.00 | 3397.90 | 26050 | 130691 |
| chromium_cdp | 1 | public_71 | warm | true | 5197.65 | 0.00 | 5197.65 | 11231 | 337989 |
| chromium_cdp | 1 | public_72 | warm | true | 1177.68 | 0.00 | 1177.68 | 11746 | 30377 |
| chromium_cdp | 1 | public_73 | warm | false | 20000.74 | 0.00 | 20000.74 | 0 | 0 |
| chromium_cdp | 1 | public_74 | warm | false | 20000.12 | 0.00 | 20000.12 | 228 | 8996 |
| chromium_cdp | 1 | public_75 | warm | true | 2086.24 | 0.00 | 2086.24 | 4374 | 7137 |
| chromium_cdp | 1 | public_76 | warm | true | 1622.48 | 0.00 | 1622.48 | 18285 | 80354 |
| chromium_cdp | 1 | public_77 | warm | true | 1711.12 | 0.00 | 1711.12 | 4814 | 5937 |
| chromium_cdp | 1 | public_78 | warm | true | 355.71 | 0.00 | 355.71 | 24551 | 333363 |
| chromium_cdp | 1 | public_79 | warm | true | 2988.31 | 0.00 | 2988.31 | 35064 | 1469700 |
| chromium_cdp | 1 | public_80 | warm | true | 3956.18 | 0.00 | 3956.18 | 37202 | 510665 |
| chromium_cdp | 1 | public_81 | warm | true | 1977.58 | 0.00 | 1977.58 | 31155 | 172954 |
| chromium_cdp | 1 | public_82 | warm | true | 980.75 | 0.00 | 980.75 | 31927 | 109978 |
| chromium_cdp | 1 | public_83 | warm | true | 2061.12 | 0.00 | 2061.12 | 58745 | 1087044 |
| chromium_cdp | 1 | public_84 | warm | true | 1758.74 | 0.00 | 1758.74 | 26924 | 556720 |
| chromium_cdp | 1 | public_85 | warm | true | 4941.28 | 0.00 | 4941.28 | 32695 | 1063888 |
| chromium_cdp | 1 | public_86 | warm | true | 1516.23 | 0.00 | 1516.23 | 29572 | 327358 |
| chromium_cdp | 1 | public_87 | warm | true | 1333.92 | 0.00 | 1333.92 | 41734 | 805267 |
| chromium_cdp | 1 | public_88 | warm | true | 1078.01 | 0.00 | 1078.01 | 40811 | 1449868 |
| chromium_cdp | 1 | public_89 | warm | true | 3412.73 | 0.00 | 3412.73 | 49564 | 2672586 |
| chromium_cdp | 1 | public_90 | warm | true | 17053.20 | 0.00 | 17053.20 | 31879 | 1694864 |
| chromium_cdp | 1 | public_91 | warm | true | 2279.26 | 0.00 | 2279.26 | 852 | 9483 |
| chromium_cdp | 1 | public_92 | warm | true | 80.24 | 0.00 | 80.24 | 787 | 723 |
| chromium_cdp | 1 | public_93 | warm | true | 3267.23 | 0.00 | 3267.23 | 18529 | 156975 |
| chromium_cdp | 1 | public_94 | warm | true | 622.74 | 0.00 | 622.74 | 1287 | 29375 |
| chromium_cdp | 1 | public_95 | warm | true | 941.69 | 0.00 | 941.69 | 20830 | 263739 |
| chromium_cdp | 1 | public_96 | warm | true | 2665.87 | 0.00 | 2665.87 | 15128 | 24828 |
| chromium_cdp | 1 | public_97 | warm | true | 1869.56 | 0.00 | 1869.56 | 15221 | 47256 |
| chromium_cdp | 1 | public_98 | warm | true | 836.60 | 0.00 | 836.60 | 18739 | 1751525 |
| chromium_cdp | 1 | public_99 | warm | true | 1462.65 | 0.00 | 1462.65 | 12451 | 261370 |
| chromium_cdp | 1 | public_100 | warm | true | 5860.44 | 0.00 | 5860.44 | 9800 | 35365 |
| chromium_cdp | 2 | public_1 | cold | true | 2103.74 | 357.37 | 2461.11 | 23816 | 316063 |
| chromium_cdp | 2 | public_2 | warm | true | 2180.94 | 0.00 | 2180.94 | 32900 | 2398492 |
| chromium_cdp | 2 | public_3 | warm | true | 1265.09 | 0.00 | 1265.09 | 34966 | 2073471 |
| chromium_cdp | 2 | public_4 | warm | true | 2274.74 | 0.00 | 2274.74 | 39013 | 3451096 |
| chromium_cdp | 2 | public_5 | warm | true | 1408.38 | 0.00 | 1408.38 | 34683 | 2217883 |
| chromium_cdp | 2 | public_6 | warm | true | 1256.66 | 0.00 | 1256.66 | 33596 | 2277895 |
| chromium_cdp | 2 | public_7 | warm | true | 2173.80 | 0.00 | 2173.80 | 55777 | 2039007 |
| chromium_cdp | 2 | public_8 | warm | true | 1880.45 | 0.00 | 1880.45 | 58494 | 3733888 |
| chromium_cdp | 2 | public_9 | warm | true | 1676.44 | 0.00 | 1676.44 | 31344 | 963891 |
| chromium_cdp | 2 | public_10 | warm | true | 1305.83 | 0.00 | 1305.83 | 32071 | 1362175 |
| chromium_cdp | 2 | public_11 | warm | true | 1544.00 | 0.00 | 1544.00 | 30281 | 643745 |
| chromium_cdp | 2 | public_12 | warm | true | 2093.82 | 0.00 | 2093.82 | 19981 | 104735 |
| chromium_cdp | 2 | public_13 | warm | true | 2385.62 | 0.00 | 2385.62 | 25375 | 240727 |
| chromium_cdp | 2 | public_14 | warm | true | 2612.03 | 0.00 | 2612.03 | 31029 | 607328 |
| chromium_cdp | 2 | public_15 | warm | true | 2080.05 | 0.00 | 2080.05 | 23210 | 212193 |
| chromium_cdp | 2 | public_16 | warm | true | 2117.69 | 0.00 | 2117.69 | 25279 | 134921 |
| chromium_cdp | 2 | public_17 | warm | true | 2188.66 | 0.00 | 2188.66 | 23266 | 153297 |
| chromium_cdp | 2 | public_18 | warm | true | 3310.59 | 0.00 | 3310.59 | 22903 | 432576 |
| chromium_cdp | 2 | public_19 | warm | true | 2065.85 | 0.00 | 2065.85 | 25703 | 93542 |
| chromium_cdp | 2 | public_20 | warm | true | 2453.34 | 0.00 | 2453.34 | 23771 | 273726 |
| chromium_cdp | 2 | public_21 | warm | true | 2439.64 | 0.00 | 2439.64 | 17008 | 62710 |
| chromium_cdp | 2 | public_22 | warm | true | 728.62 | 0.00 | 728.62 | 10807 | 35845 |
| chromium_cdp | 2 | public_23 | warm | true | 3165.97 | 0.00 | 3165.97 | 17645 | 62661 |
| chromium_cdp | 2 | public_24 | warm | true | 1113.32 | 0.00 | 1113.32 | 21865 | 34094 |
| chromium_cdp | 2 | public_25 | warm | true | 256.22 | 0.00 | 256.22 | 24531 | 42014 |
| chromium_cdp | 2 | public_26 | warm | false | 266.03 | 0.00 | 266.03 | 773 | 28497 |
| chromium_cdp | 2 | public_27 | warm | false | 271.96 | 0.00 | 271.96 | 757 | 28481 |
| chromium_cdp | 2 | public_28 | warm | false | 252.24 | 0.00 | 252.24 | 765 | 28489 |
| chromium_cdp | 2 | public_29 | warm | false | 427.67 | 0.00 | 427.67 | 757 | 28481 |
| chromium_cdp | 2 | public_30 | warm | true | 2570.91 | 0.00 | 2570.91 | 19643 | 611022 |
| chromium_cdp | 2 | public_31 | warm | true | 1311.67 | 0.00 | 1311.67 | 18255 | 301465 |
| chromium_cdp | 2 | public_32 | warm | true | 398.53 | 0.00 | 398.53 | 12112 | 24203 |
| chromium_cdp | 2 | public_33 | warm | true | 303.34 | 0.00 | 303.34 | 6026 | 17486 |
| chromium_cdp | 2 | public_34 | warm | true | 307.77 | 0.00 | 307.77 | 7364 | 18417 |
| chromium_cdp | 2 | public_35 | warm | true | 358.85 | 0.00 | 358.85 | 27811 | 139965 |
| chromium_cdp | 2 | public_36 | warm | true | 210.75 | 0.00 | 210.75 | 22227 | 75798 |
| chromium_cdp | 2 | public_37 | warm | true | 3426.40 | 0.00 | 3426.40 | 8406 | 20740 |
| chromium_cdp | 2 | public_38 | warm | true | 6791.10 | 0.00 | 6791.10 | 23536 | 132519 |
| chromium_cdp | 2 | public_39 | warm | false | 521.79 | 0.00 | 521.79 | 745 | 28469 |
| chromium_cdp | 2 | public_40 | warm | true | 3136.53 | 0.00 | 3136.53 | 39823 | 595741 |
| chromium_cdp | 2 | public_41 | warm | true | 338.65 | 0.00 | 338.65 | 9812 | 258787 |
| chromium_cdp | 2 | public_42 | warm | true | 381.29 | 0.00 | 381.29 | 8991 | 531376 |
| chromium_cdp | 2 | public_43 | warm | true | 6935.82 | 0.00 | 6935.82 | 23351 | 328509 |
| chromium_cdp | 2 | public_44 | warm | true | 2595.95 | 0.00 | 2595.95 | 14381 | 64904 |
| chromium_cdp | 2 | public_45 | warm | true | 836.64 | 0.00 | 836.64 | 9891 | 12458 |
| chromium_cdp | 2 | public_46 | warm | true | 2484.06 | 0.00 | 2484.06 | 13351 | 61610 |
| chromium_cdp | 2 | public_47 | warm | true | 5714.94 | 0.00 | 5714.94 | 18722 | 694697 |
| chromium_cdp | 2 | public_48 | warm | true | 2384.32 | 0.00 | 2384.32 | 9664 | 400360 |
| chromium_cdp | 2 | public_49 | warm | true | 3509.32 | 0.00 | 3509.32 | 30318 | 330195 |
| chromium_cdp | 2 | public_50 | warm | true | 643.39 | 0.00 | 643.39 | 6144 | 15517 |
| chromium_cdp | 2 | public_51 | warm | true | 349.89 | 0.00 | 349.89 | 9525 | 19044 |
| chromium_cdp | 2 | public_52 | warm | false | 20000.17 | 0.00 | 20000.17 | 0 | 0 |
| chromium_cdp | 2 | public_53 | warm | false | 1574.25 | 0.00 | 1574.25 | 263 | 8620 |
| chromium_cdp | 2 | public_54 | warm | true | 1923.27 | 0.00 | 1923.27 | 19937 | 52939 |
| chromium_cdp | 2 | public_55 | warm | true | 3630.21 | 0.00 | 3630.21 | 24060 | 52874 |
| chromium_cdp | 2 | public_56 | warm | false | 20000.37 | 0.00 | 20000.37 | 0 | 0 |
| chromium_cdp | 2 | public_57 | warm | true | 1929.19 | 0.00 | 1929.19 | 12096 | 193151 |
| chromium_cdp | 2 | public_58 | warm | true | 848.94 | 0.00 | 848.94 | 1443 | 6242 |
| chromium_cdp | 2 | public_59 | warm | true | 143.71 | 0.00 | 143.71 | 2851 | 2358 |
| chromium_cdp | 2 | public_60 | warm | true | 735.07 | 0.00 | 735.07 | 926 | 646 |
| chromium_cdp | 2 | public_61 | warm | true | 249.09 | 0.00 | 249.09 | 21392 | 37786 |
| chromium_cdp | 2 | public_62 | warm | true | 267.56 | 0.00 | 267.56 | 21833 | 100773 |
| chromium_cdp | 2 | public_63 | warm | true | 978.25 | 0.00 | 978.25 | 29317 | 397301 |
| chromium_cdp | 2 | public_64 | warm | true | 884.90 | 0.00 | 884.90 | 7122 | 181523 |
| chromium_cdp | 2 | public_65 | warm | true | 1933.46 | 0.00 | 1933.46 | 4334 | 17828 |
| chromium_cdp | 2 | public_66 | warm | true | 315.42 | 0.00 | 315.42 | 28883 | 77867 |
| chromium_cdp | 2 | public_67 | warm | true | 3373.67 | 0.00 | 3373.67 | 9455 | 49907 |
| chromium_cdp | 2 | public_68 | warm | true | 1916.59 | 0.00 | 1916.59 | 11873 | 50418 |
| chromium_cdp | 2 | public_69 | warm | false | 20000.10 | 0.00 | 20000.10 | 0 | 0 |
| chromium_cdp | 2 | public_70 | warm | true | 3789.23 | 0.00 | 3789.23 | 26050 | 130691 |
| chromium_cdp | 2 | public_71 | warm | true | 4923.16 | 0.00 | 4923.16 | 11231 | 338125 |
| chromium_cdp | 2 | public_72 | warm | true | 1203.16 | 0.00 | 1203.16 | 11746 | 30377 |
| chromium_cdp | 2 | public_73 | warm | false | 20000.04 | 0.00 | 20000.04 | 0 | 0 |
| chromium_cdp | 2 | public_74 | warm | false | 20000.24 | 0.00 | 20000.24 | 228 | 8996 |
| chromium_cdp | 2 | public_75 | warm | true | 416.77 | 0.00 | 416.77 | 4374 | 7137 |
| chromium_cdp | 2 | public_76 | warm | true | 2096.60 | 0.00 | 2096.60 | 18285 | 80354 |
| chromium_cdp | 2 | public_77 | warm | true | 426.11 | 0.00 | 426.11 | 4814 | 5937 |
| chromium_cdp | 2 | public_78 | warm | true | 352.55 | 0.00 | 352.55 | 24551 | 333363 |
| chromium_cdp | 2 | public_79 | warm | true | 3161.23 | 0.00 | 3161.23 | 35064 | 1469668 |
| chromium_cdp | 2 | public_80 | warm | true | 3539.35 | 0.00 | 3539.35 | 37202 | 511108 |
| chromium_cdp | 2 | public_81 | warm | true | 1447.47 | 0.00 | 1447.47 | 31365 | 173163 |
| chromium_cdp | 2 | public_82 | warm | true | 1501.47 | 0.00 | 1501.47 | 31927 | 109978 |
| chromium_cdp | 2 | public_83 | warm | true | 2030.68 | 0.00 | 2030.68 | 58745 | 1087044 |
| chromium_cdp | 2 | public_84 | warm | true | 2304.19 | 0.00 | 2304.19 | 26934 | 556439 |
| chromium_cdp | 2 | public_85 | warm | true | 4893.97 | 0.00 | 4893.97 | 32627 | 1069721 |
| chromium_cdp | 2 | public_86 | warm | true | 1477.71 | 0.00 | 1477.71 | 29572 | 327358 |
| chromium_cdp | 2 | public_87 | warm | false | 20000.19 | 0.00 | 20000.19 | 0 | 0 |
| chromium_cdp | 2 | public_88 | warm | true | 976.71 | 0.00 | 976.71 | 40811 | 1450844 |
| chromium_cdp | 2 | public_89 | warm | true | 3753.61 | 0.00 | 3753.61 | 49604 | 2686635 |
| chromium_cdp | 2 | public_90 | warm | true | 17323.41 | 0.00 | 17323.41 | 31879 | 1694842 |
| chromium_cdp | 2 | public_91 | warm | true | 2242.39 | 0.00 | 2242.39 | 852 | 9483 |
| chromium_cdp | 2 | public_92 | warm | true | 235.22 | 0.00 | 235.22 | 787 | 723 |
| chromium_cdp | 2 | public_93 | warm | true | 3169.46 | 0.00 | 3169.46 | 18254 | 151446 |
| chromium_cdp | 2 | public_94 | warm | true | 1184.82 | 0.00 | 1184.82 | 1287 | 29375 |
| chromium_cdp | 2 | public_95 | warm | true | 1313.90 | 0.00 | 1313.90 | 20830 | 263739 |
| chromium_cdp | 2 | public_96 | warm | true | 2376.03 | 0.00 | 2376.03 | 15128 | 24828 |
| chromium_cdp | 2 | public_97 | warm | true | 2357.12 | 0.00 | 2357.12 | 15221 | 47255 |
| chromium_cdp | 2 | public_98 | warm | true | 2683.46 | 0.00 | 2683.46 | 19523 | 1807610 |
| chromium_cdp | 2 | public_99 | warm | true | 2296.09 | 0.00 | 2296.09 | 12451 | 261370 |
| chromium_cdp | 2 | public_100 | warm | true | 3029.64 | 0.00 | 3029.64 | 9800 | 35365 |
| ai_go | 2 | public_1 | cold | true | 1111.82 | 0.00 | 1111.82 | 29611 | 253943 |
| ai_go | 2 | public_2 | warm | true | 2315.25 | 0.00 | 2315.25 | 33849 | 2017450 |
| ai_go | 2 | public_3 | warm | true | 1925.52 | 0.00 | 1925.52 | 34992 | 1876166 |
| ai_go | 2 | public_4 | warm | true | 2078.68 | 0.00 | 2078.68 | 37329 | 2873942 |
| ai_go | 2 | public_5 | warm | true | 1947.91 | 0.00 | 1947.91 | 36305 | 2075994 |
| ai_go | 2 | public_6 | warm | true | 1802.02 | 0.00 | 1802.02 | 34924 | 2113993 |
| ai_go | 2 | public_7 | warm | true | 1860.11 | 0.00 | 1860.11 | 38034 | 1866457 |
| ai_go | 2 | public_8 | warm | true | 2118.73 | 0.00 | 2118.73 | 41138 | 3249113 |
| ai_go | 2 | public_9 | warm | true | 1657.21 | 0.00 | 1657.21 | 34305 | 864585 |
| ai_go | 2 | public_10 | warm | true | 1615.49 | 0.00 | 1615.49 | 33604 | 1282423 |
| ai_go | 2 | public_11 | warm | true | 1359.11 | 0.00 | 1359.11 | 32245 | 254179 |
| ai_go | 2 | public_12 | warm | true | 1198.70 | 0.00 | 1198.70 | 27573 | 102830 |
| ai_go | 2 | public_13 | warm | true | 1131.20 | 0.00 | 1131.20 | 28417 | 174324 |
| ai_go | 2 | public_14 | warm | true | 1233.34 | 0.00 | 1233.34 | 37086 | 127227 |
| ai_go | 2 | public_15 | warm | true | 1487.86 | 0.00 | 1487.86 | 27983 | 156046 |
| ai_go | 2 | public_16 | warm | true | 1487.10 | 0.00 | 1487.10 | 31502 | 132580 |
| ai_go | 2 | public_17 | warm | true | 1460.09 | 0.00 | 1460.09 | 28243 | 95737 |
| ai_go | 2 | public_18 | warm | true | 1361.18 | 0.00 | 1361.18 | 28722 | 135080 |
| ai_go | 2 | public_19 | warm | true | 1307.42 | 0.00 | 1307.42 | 28738 | 91459 |
| ai_go | 2 | public_20 | warm | true | 1312.62 | 0.00 | 1312.62 | 30696 | 213319 |
| ai_go | 2 | public_21 | warm | true | 695.37 | 0.00 | 695.37 | 20425 | 64188 |
| ai_go | 2 | public_22 | warm | true | 233.45 | 0.00 | 233.45 | 11811 | 34115 |
| ai_go | 2 | public_23 | warm | true | 1648.03 | 0.00 | 1648.03 | 21057 | 64188 |
| ai_go | 2 | public_24 | warm | true | 720.04 | 0.00 | 720.04 | 24595 | 34040 |
| ai_go | 2 | public_25 | warm | true | 196.81 | 0.00 | 196.81 | 28284 | 40740 |
| ai_go | 2 | public_26 | warm | false | 20000.73 | 0.00 | 20000.73 | 317 | 5653 |
| ai_go | 2 | public_27 | warm | false | 20000.27 | 0.00 | 20000.27 | 313 | 5649 |
| ai_go | 2 | public_28 | warm | false | 20000.42 | 0.00 | 20000.42 | 315 | 5651 |
| ai_go | 2 | public_29 | warm | false | 20000.22 | 0.00 | 20000.22 | 313 | 5649 |
| ai_go | 2 | public_30 | warm | true | 2258.81 | 0.00 | 2258.81 | 28864 | 608683 |
| ai_go | 2 | public_31 | warm | true | 615.42 | 0.00 | 615.42 | 27325 | 300785 |
| ai_go | 2 | public_32 | warm | true | 378.78 | 0.00 | 378.78 | 14226 | 19625 |
| ai_go | 2 | public_33 | warm | true | 287.86 | 0.00 | 287.86 | 7702 | 20026 |
| ai_go | 2 | public_34 | warm | true | 618.90 | 0.00 | 618.90 | 9550 | 18594 |
| ai_go | 2 | public_35 | warm | true | 68.72 | 0.00 | 68.72 | 32345 | 222341 |
| ai_go | 2 | public_36 | warm | true | 30.39 | 0.00 | 30.39 | 25000 | 121746 |
| ai_go | 2 | public_37 | warm | true | 1144.35 | 0.00 | 1144.35 | 9882 | 19190 |
| ai_go | 2 | public_38 | warm | true | 425.22 | 0.00 | 425.22 | 19678 | 132701 |
| ai_go | 2 | public_39 | warm | false | 20000.53 | 0.00 | 20000.53 | 310 | 5646 |
| ai_go | 2 | public_40 | warm | true | 102.14 | 0.00 | 102.14 | 34229 | 372981 |
| ai_go | 2 | public_41 | warm | true | 65.24 | 0.00 | 65.24 | 11937 | 258828 |
| ai_go | 2 | public_42 | warm | true | 181.36 | 0.00 | 181.36 | 13755 | 531745 |
| ai_go | 2 | public_43 | warm | true | 2716.40 | 0.00 | 2716.40 | 26858 | 289647 |
| ai_go | 2 | public_44 | warm | true | 1611.53 | 0.00 | 1611.53 | 17076 | 59911 |
| ai_go | 2 | public_45 | warm | true | 268.18 | 0.00 | 268.18 | 11297 | 12617 |
| ai_go | 2 | public_46 | warm | true | 45.05 | 0.00 | 45.05 | 15530 | 41608 |
| ai_go | 2 | public_47 | warm | true | 572.05 | 0.00 | 572.05 | 21407 | 500979 |
| ai_go | 2 | public_48 | warm | true | 1052.40 | 0.00 | 1052.40 | 10775 | 185401 |
| ai_go | 2 | public_49 | warm | true | 1106.33 | 0.00 | 1106.33 | 44520 | 209845 |
| ai_go | 2 | public_50 | warm | true | 117.49 | 0.00 | 117.49 | 8333 | 15864 |
| ai_go | 2 | public_51 | warm | true | 741.48 | 0.00 | 741.48 | 10866 | 19150 |
| ai_go | 2 | public_52 | warm | false | 10000.98 | 0.00 | 10000.98 | 0 | 0 |
| ai_go | 2 | public_53 | warm | false | 20000.86 | 0.00 | 20000.86 | 306 | 8293 |
| ai_go | 2 | public_54 | warm | true | 367.84 | 0.00 | 367.84 | 21504 | 51923 |
| ai_go | 2 | public_55 | warm | false | 10376.60 | 0.00 | 10376.60 | 0 | 0 |
| ai_go | 2 | public_56 | warm | true | 804.34 | 0.00 | 804.34 | 339 | 1145 |
| ai_go | 2 | public_57 | warm | true | 646.43 | 0.00 | 646.43 | 14484 | 192265 |
| ai_go | 2 | public_58 | warm | true | 15769.71 | 0.00 | 15769.71 | 1906 | 6688 |
| ai_go | 2 | public_59 | warm | true | 221.72 | 0.00 | 221.72 | 1701 | 2394 |
| ai_go | 2 | public_60 | warm | true | 808.07 | 0.00 | 808.07 | 1078 | 755 |
| ai_go | 2 | public_61 | warm | true | 283.65 | 0.00 | 283.65 | 24849 | 44341 |
| ai_go | 2 | public_62 | warm | true | 255.33 | 0.00 | 255.33 | 28325 | 117674 |
| ai_go | 2 | public_63 | warm | true | 920.97 | 0.00 | 920.97 | 46205 | 432718 |
| ai_go | 2 | public_64 | warm | true | 838.51 | 0.00 | 838.51 | 8857 | 183802 |
| ai_go | 2 | public_65 | warm | true | 2156.54 | 0.00 | 2156.54 | 4952 | 19304 |
| ai_go | 2 | public_66 | warm | true | 366.25 | 0.00 | 366.25 | 11000 | 83835 |
| ai_go | 2 | public_67 | warm | true | 3271.28 | 0.00 | 3271.28 | 12300 | 52338 |
| ai_go | 2 | public_68 | warm | true | 1307.99 | 0.00 | 1307.99 | 13505 | 52894 |
| ai_go | 2 | public_69 | warm | false | 20000.38 | 0.00 | 20000.38 | 0 | 0 |
| ai_go | 2 | public_70 | warm | true | 4873.75 | 0.00 | 4873.75 | 24477 | 147230 |
| ai_go | 2 | public_71 | warm | true | 5270.17 | 0.00 | 5270.17 | 15893 | 344807 |
| ai_go | 2 | public_72 | warm | true | 1257.24 | 0.00 | 1257.24 | 13324 | 33081 |
| ai_go | 2 | public_73 | warm | false | 20006.19 | 0.00 | 20006.19 | 0 | 0 |
| ai_go | 2 | public_74 | warm | true | 1366.65 | 0.00 | 1366.65 | 2739 | 77046 |
| ai_go | 2 | public_75 | warm | true | 1991.07 | 0.00 | 1991.07 | 5396 | 8237 |
| ai_go | 2 | public_76 | warm | true | 1768.75 | 0.00 | 1768.75 | 21716 | 100312 |
| ai_go | 2 | public_77 | warm | true | 1894.64 | 0.00 | 1894.64 | 5259 | 6559 |
| ai_go | 2 | public_78 | warm | true | 1047.66 | 0.00 | 1047.66 | 26708 | 335708 |
| ai_go | 2 | public_79 | warm | true | 5937.70 | 0.00 | 5937.70 | 60481 | 1492075 |
| ai_go | 2 | public_80 | warm | true | 4445.46 | 0.00 | 4445.46 | 46226 | 521035 |
| ai_go | 2 | public_81 | warm | true | 1643.70 | 0.00 | 1643.70 | 35952 | 239472 |
| ai_go | 2 | public_82 | warm | true | 1257.39 | 0.00 | 1257.39 | 34687 | 115627 |
| ai_go | 2 | public_83 | warm | true | 3061.65 | 0.00 | 3061.65 | 69766 | 1009824 |
| ai_go | 2 | public_84 | warm | true | 2391.44 | 0.00 | 2391.44 | 28982 | 563150 |
| ai_go | 2 | public_85 | warm | true | 5609.36 | 0.00 | 5609.36 | 42370 | 1115456 |
| ai_go | 2 | public_86 | warm | true | 2070.63 | 0.00 | 2070.63 | 42593 | 337670 |
| ai_go | 2 | public_87 | warm | true | 2751.89 | 0.00 | 2751.89 | 60550 | 603534 |
| ai_go | 2 | public_88 | warm | true | 1203.52 | 0.00 | 1203.52 | 58274 | 1477296 |
| ai_go | 2 | public_89 | warm | true | 4404.48 | 0.00 | 4404.48 | 69534 | 1789608 |
| ai_go | 2 | public_90 | warm | true | 16753.32 | 0.00 | 16753.32 | 36666 | 1703216 |
| ai_go | 2 | public_91 | warm | true | 2221.61 | 0.00 | 2221.61 | 937 | 9603 |
| ai_go | 2 | public_92 | warm | true | 288.60 | 0.00 | 288.60 | 905 | 738 |
| ai_go | 2 | public_93 | warm | true | 9785.29 | 0.00 | 9785.29 | 22453 | 154132 |
| ai_go | 2 | public_94 | warm | true | 732.06 | 0.00 | 732.06 | 3043 | 31237 |
| ai_go | 2 | public_95 | warm | true | 4860.44 | 0.00 | 4860.44 | 22780 | 276107 |
| ai_go | 2 | public_96 | warm | false | 12540.27 | 0.00 | 12540.27 | 0 | 0 |
| ai_go | 2 | public_97 | warm | true | 3004.83 | 0.00 | 3004.83 | 14196 | 51470 |
| ai_go | 2 | public_98 | warm | true | 2780.58 | 0.00 | 2780.58 | 16336 | 1821708 |
| ai_go | 2 | public_99 | warm | true | 4145.86 | 0.00 | 4145.86 | 13549 | 265336 |
| ai_go | 2 | public_100 | warm | true | 3036.06 | 0.00 | 3036.06 | 11024 | 37267 |
| ai_go | 3 | public_1 | cold | true | 1111.29 | 0.00 | 1111.29 | 29611 | 254005 |
| ai_go | 3 | public_2 | warm | true | 2202.49 | 0.00 | 2202.49 | 33849 | 2017852 |
| ai_go | 3 | public_3 | warm | true | 2335.46 | 0.00 | 2335.46 | 34992 | 1876568 |
| ai_go | 3 | public_4 | warm | true | 2272.79 | 0.00 | 2272.79 | 37329 | 2874284 |
| ai_go | 3 | public_5 | warm | true | 2600.15 | 0.00 | 2600.15 | 36305 | 2076254 |
| ai_go | 3 | public_6 | warm | true | 2337.47 | 0.00 | 2337.47 | 34924 | 2114395 |
| ai_go | 3 | public_7 | warm | true | 2601.24 | 0.00 | 2601.24 | 38034 | 1866861 |
| ai_go | 3 | public_8 | warm | true | 2124.51 | 0.00 | 2124.51 | 41138 | 3249113 |
| ai_go | 3 | public_9 | warm | true | 1517.99 | 0.00 | 1517.99 | 34305 | 864585 |
| ai_go | 3 | public_10 | warm | true | 2192.39 | 0.00 | 2192.39 | 33604 | 1282631 |
| ai_go | 3 | public_11 | warm | true | 1308.07 | 0.00 | 1308.07 | 32245 | 254179 |
| ai_go | 3 | public_12 | warm | true | 919.38 | 0.00 | 919.38 | 27573 | 102830 |
| ai_go | 3 | public_13 | warm | true | 1219.08 | 0.00 | 1219.08 | 28417 | 174324 |
| ai_go | 3 | public_14 | warm | true | 1382.20 | 0.00 | 1382.20 | 37086 | 127227 |
| ai_go | 3 | public_15 | warm | true | 1078.30 | 0.00 | 1078.30 | 27983 | 156046 |
| ai_go | 3 | public_16 | warm | true | 1109.33 | 0.00 | 1109.33 | 31502 | 132580 |
| ai_go | 3 | public_17 | warm | true | 1195.08 | 0.00 | 1195.08 | 28243 | 95737 |
| ai_go | 3 | public_18 | warm | true | 1554.30 | 0.00 | 1554.30 | 28722 | 135080 |
| ai_go | 3 | public_19 | warm | true | 1497.22 | 0.00 | 1497.22 | 28738 | 91459 |
| ai_go | 3 | public_20 | warm | true | 1786.92 | 0.00 | 1786.92 | 30696 | 213319 |
| ai_go | 3 | public_21 | warm | true | 625.76 | 0.00 | 625.76 | 20425 | 64188 |
| ai_go | 3 | public_22 | warm | true | 263.26 | 0.00 | 263.26 | 11811 | 34115 |
| ai_go | 3 | public_23 | warm | true | 1557.05 | 0.00 | 1557.05 | 21057 | 64188 |
| ai_go | 3 | public_24 | warm | true | 728.70 | 0.00 | 728.70 | 24615 | 34130 |
| ai_go | 3 | public_25 | warm | true | 235.31 | 0.00 | 235.31 | 28123 | 40684 |
| ai_go | 3 | public_26 | warm | false | 20000.11 | 0.00 | 20000.11 | 317 | 5653 |
| ai_go | 3 | public_27 | warm | false | 20001.38 | 0.00 | 20001.38 | 313 | 5649 |
| ai_go | 3 | public_28 | warm | false | 20000.64 | 0.00 | 20000.64 | 315 | 5651 |
| ai_go | 3 | public_29 | warm | false | 20000.42 | 0.00 | 20000.42 | 313 | 5649 |
| ai_go | 3 | public_30 | warm | true | 2304.05 | 0.00 | 2304.05 | 28868 | 608687 |
| ai_go | 3 | public_31 | warm | true | 615.44 | 0.00 | 615.44 | 27325 | 300785 |
| ai_go | 3 | public_32 | warm | true | 376.48 | 0.00 | 376.48 | 14226 | 19625 |
| ai_go | 3 | public_33 | warm | true | 589.13 | 0.00 | 589.13 | 7368 | 19602 |
| ai_go | 3 | public_34 | warm | true | 484.59 | 0.00 | 484.59 | 9550 | 18594 |
| ai_go | 3 | public_35 | warm | true | 99.67 | 0.00 | 99.67 | 32345 | 222341 |
| ai_go | 3 | public_36 | warm | true | 33.42 | 0.00 | 33.42 | 25000 | 121746 |
| ai_go | 3 | public_37 | warm | true | 1073.22 | 0.00 | 1073.22 | 9882 | 19190 |
| ai_go | 3 | public_38 | warm | true | 501.84 | 0.00 | 501.84 | 19678 | 132697 |
| ai_go | 3 | public_39 | warm | false | 20000.31 | 0.00 | 20000.31 | 310 | 5646 |
| ai_go | 3 | public_40 | warm | true | 120.86 | 0.00 | 120.86 | 34229 | 372981 |
| ai_go | 3 | public_41 | warm | true | 61.36 | 0.00 | 61.36 | 11937 | 258828 |
| ai_go | 3 | public_42 | warm | true | 329.43 | 0.00 | 329.43 | 13755 | 531732 |
| ai_go | 3 | public_43 | warm | true | 2728.91 | 0.00 | 2728.91 | 26858 | 289647 |
| ai_go | 3 | public_44 | warm | true | 2071.93 | 0.00 | 2071.93 | 17076 | 59911 |
| ai_go | 3 | public_45 | warm | true | 767.61 | 0.00 | 767.61 | 11297 | 12617 |
| ai_go | 3 | public_46 | warm | true | 47.07 | 0.00 | 47.07 | 15530 | 41608 |
| ai_go | 3 | public_47 | warm | true | 1049.49 | 0.00 | 1049.49 | 21407 | 500979 |
| ai_go | 3 | public_48 | warm | true | 990.01 | 0.00 | 990.01 | 10775 | 185401 |
| ai_go | 3 | public_49 | warm | true | 1063.51 | 0.00 | 1063.51 | 44520 | 209845 |
| ai_go | 3 | public_50 | warm | true | 329.60 | 0.00 | 329.60 | 8333 | 15864 |
| ai_go | 3 | public_51 | warm | true | 366.19 | 0.00 | 366.19 | 10866 | 19150 |
| ai_go | 3 | public_52 | warm | false | 10000.48 | 0.00 | 10000.48 | 0 | 0 |
| ai_go | 3 | public_53 | warm | false | 20000.54 | 0.00 | 20000.54 | 306 | 8293 |
| ai_go | 3 | public_54 | warm | true | 79.60 | 0.00 | 79.60 | 21504 | 51923 |
| ai_go | 3 | public_55 | warm | true | 1039.57 | 0.00 | 1039.57 | 27669 | 53921 |
| ai_go | 3 | public_56 | warm | true | 342.81 | 0.00 | 342.81 | 339 | 1145 |
| ai_go | 3 | public_57 | warm | true | 64.53 | 0.00 | 64.53 | 14484 | 192265 |
| ai_go | 3 | public_58 | warm | true | 1742.81 | 0.00 | 1742.81 | 1906 | 6688 |
| ai_go | 3 | public_59 | warm | true | 223.55 | 0.00 | 223.55 | 1701 | 2394 |
| ai_go | 3 | public_60 | warm | true | 855.23 | 0.00 | 855.23 | 1078 | 755 |
| ai_go | 3 | public_61 | warm | true | 289.10 | 0.00 | 289.10 | 24849 | 44341 |
| ai_go | 3 | public_62 | warm | true | 621.46 | 0.00 | 621.46 | 28325 | 117674 |
| ai_go | 3 | public_63 | warm | true | 1068.74 | 0.00 | 1068.74 | 46205 | 432718 |
| ai_go | 3 | public_64 | warm | true | 790.64 | 0.00 | 790.64 | 8857 | 183811 |
| ai_go | 3 | public_65 | warm | true | 2059.50 | 0.00 | 2059.50 | 4952 | 19304 |
| ai_go | 3 | public_66 | warm | true | 382.11 | 0.00 | 382.11 | 11000 | 83835 |
| ai_go | 3 | public_67 | warm | true | 3956.59 | 0.00 | 3956.59 | 12300 | 52336 |
| ai_go | 3 | public_68 | warm | true | 1324.12 | 0.00 | 1324.12 | 13505 | 52892 |
| ai_go | 3 | public_69 | warm | true | 7569.11 | 0.00 | 7569.11 | 21922 | 358122 |
| ai_go | 3 | public_70 | warm | true | 7257.87 | 0.00 | 7257.87 | 24477 | 147262 |
| ai_go | 3 | public_71 | warm | true | 5236.74 | 0.00 | 5236.74 | 15893 | 344807 |
| ai_go | 3 | public_72 | warm | true | 1260.05 | 0.00 | 1260.05 | 13324 | 33081 |
| ai_go | 3 | public_73 | warm | false | 20005.57 | 0.00 | 20005.57 | 0 | 0 |
| ai_go | 3 | public_74 | warm | true | 1334.90 | 0.00 | 1334.90 | 2739 | 77046 |
| ai_go | 3 | public_75 | warm | true | 2010.64 | 0.00 | 2010.64 | 5396 | 8237 |
| ai_go | 3 | public_76 | warm | true | 1767.13 | 0.00 | 1767.13 | 21716 | 100312 |
| ai_go | 3 | public_77 | warm | true | 563.73 | 0.00 | 563.73 | 5259 | 6559 |
| ai_go | 3 | public_78 | warm | true | 318.88 | 0.00 | 318.88 | 26708 | 335708 |
| ai_go | 3 | public_79 | warm | true | 5932.21 | 0.00 | 5932.21 | 60481 | 1492088 |
| ai_go | 3 | public_80 | warm | true | 3788.83 | 0.00 | 3788.83 | 46226 | 521035 |
| ai_go | 3 | public_81 | warm | true | 1908.08 | 0.00 | 1908.08 | 35905 | 238896 |
| ai_go | 3 | public_82 | warm | true | 1154.86 | 0.00 | 1154.86 | 34687 | 115627 |
| ai_go | 3 | public_83 | warm | true | 3857.44 | 0.00 | 3857.44 | 75212 | 1178897 |
| ai_go | 3 | public_84 | warm | true | 1936.41 | 0.00 | 1936.41 | 28996 | 564109 |
| ai_go | 3 | public_85 | warm | true | 5731.78 | 0.00 | 5731.78 | 42143 | 1115044 |
| ai_go | 3 | public_86 | warm | true | 1653.30 | 0.00 | 1653.30 | 42593 | 335275 |
| ai_go | 3 | public_87 | warm | true | 4716.01 | 0.00 | 4716.01 | 60530 | 603687 |
| ai_go | 3 | public_88 | warm | true | 1382.92 | 0.00 | 1382.92 | 58274 | 1456130 |
| ai_go | 3 | public_89 | warm | true | 6015.01 | 0.00 | 6015.01 | 69712 | 1788975 |
| ai_go | 3 | public_90 | warm | true | 17237.81 | 0.00 | 17237.81 | 36993 | 1703965 |
| ai_go | 3 | public_91 | warm | true | 2317.59 | 0.00 | 2317.59 | 937 | 9603 |
| ai_go | 3 | public_92 | warm | true | 289.06 | 0.00 | 289.06 | 905 | 738 |
| ai_go | 3 | public_93 | warm | true | 3200.38 | 0.00 | 3200.38 | 22297 | 150606 |
| ai_go | 3 | public_94 | warm | true | 1305.26 | 0.00 | 1305.26 | 3043 | 31237 |
| ai_go | 3 | public_95 | warm | true | 1500.43 | 0.00 | 1500.43 | 22780 | 276107 |
| ai_go | 3 | public_96 | warm | true | 2446.12 | 0.00 | 2446.12 | 17366 | 28321 |
| ai_go | 3 | public_97 | warm | true | 1842.98 | 0.00 | 1842.98 | 14197 | 51472 |
| ai_go | 3 | public_98 | warm | true | 3603.63 | 0.00 | 3603.63 | 16336 | 1821734 |
| ai_go | 3 | public_99 | warm | true | 2429.38 | 0.00 | 2429.38 | 13549 | 265318 |
| ai_go | 3 | public_100 | warm | true | 3289.05 | 0.00 | 3289.05 | 11024 | 37267 |
| chromium_cdp | 3 | public_1 | cold | true | 2392.06 | 407.17 | 2799.23 | 23816 | 316062 |
| chromium_cdp | 3 | public_2 | warm | true | 1440.96 | 0.00 | 1440.96 | 31834 | 2202682 |
| chromium_cdp | 3 | public_3 | warm | true | 1325.89 | 0.00 | 1325.89 | 34966 | 2014471 |
| chromium_cdp | 3 | public_4 | warm | true | 2231.26 | 0.00 | 2231.26 | 39013 | 3451096 |
| chromium_cdp | 3 | public_5 | warm | true | 1327.22 | 0.00 | 1327.22 | 34683 | 2217883 |
| chromium_cdp | 3 | public_6 | warm | true | 1522.07 | 0.00 | 1522.07 | 33596 | 2277895 |
| chromium_cdp | 3 | public_7 | warm | true | 2034.46 | 0.00 | 2034.46 | 55777 | 2039007 |
| chromium_cdp | 3 | public_8 | warm | true | 1970.19 | 0.00 | 1970.19 | 58494 | 3733888 |
| chromium_cdp | 3 | public_9 | warm | true | 1525.96 | 0.00 | 1525.96 | 31344 | 963891 |
| chromium_cdp | 3 | public_10 | warm | true | 1234.29 | 0.00 | 1234.29 | 32071 | 1362175 |
| chromium_cdp | 3 | public_11 | warm | true | 2451.26 | 0.00 | 2451.26 | 24942 | 328680 |
| chromium_cdp | 3 | public_12 | warm | true | 2646.44 | 0.00 | 2646.44 | 21176 | 186639 |
| chromium_cdp | 3 | public_13 | warm | true | 2160.37 | 0.00 | 2160.37 | 25375 | 177598 |
| chromium_cdp | 3 | public_14 | warm | true | 2763.39 | 0.00 | 2763.39 | 31029 | 607328 |
| chromium_cdp | 3 | public_15 | warm | true | 2223.09 | 0.00 | 2223.09 | 23210 | 212193 |
| chromium_cdp | 3 | public_16 | warm | false | 1641.62 | 0.00 | 1641.62 | 685 | 2115 |
| chromium_cdp | 3 | public_17 | warm | true | 2751.72 | 0.00 | 2751.72 | 24468 | 436512 |
| chromium_cdp | 3 | public_18 | warm | true | 3489.78 | 0.00 | 3489.78 | 22903 | 432576 |
| chromium_cdp | 3 | public_19 | warm | true | 2038.48 | 0.00 | 2038.48 | 25703 | 93542 |
| chromium_cdp | 3 | public_20 | warm | true | 2441.15 | 0.00 | 2441.15 | 23771 | 273726 |
| chromium_cdp | 3 | public_21 | warm | true | 2689.31 | 0.00 | 2689.31 | 17008 | 62710 |
| chromium_cdp | 3 | public_22 | warm | true | 730.86 | 0.00 | 730.86 | 10755 | 35845 |
| chromium_cdp | 3 | public_23 | warm | true | 3537.64 | 0.00 | 3537.64 | 17645 | 62661 |
| chromium_cdp | 3 | public_24 | warm | true | 992.75 | 0.00 | 992.75 | 21879 | 34166 |
| chromium_cdp | 3 | public_25 | warm | true | 242.34 | 0.00 | 242.34 | 24560 | 41543 |
| chromium_cdp | 3 | public_26 | warm | false | 416.95 | 0.00 | 416.95 | 773 | 28497 |
| chromium_cdp | 3 | public_27 | warm | false | 445.04 | 0.00 | 445.04 | 757 | 28481 |
| chromium_cdp | 3 | public_28 | warm | false | 261.48 | 0.00 | 261.48 | 765 | 28489 |
| chromium_cdp | 3 | public_29 | warm | true | 2976.02 | 0.00 | 2976.02 | 27163 | 245716 |
| chromium_cdp | 3 | public_30 | warm | true | 2858.71 | 0.00 | 2858.71 | 19643 | 611022 |
| chromium_cdp | 3 | public_31 | warm | true | 1361.57 | 0.00 | 1361.57 | 18255 | 301465 |
| chromium_cdp | 3 | public_32 | warm | true | 527.67 | 0.00 | 527.67 | 12112 | 24203 |
| chromium_cdp | 3 | public_33 | warm | true | 476.51 | 0.00 | 476.51 | 8184 | 19603 |
| chromium_cdp | 3 | public_34 | warm | true | 391.14 | 0.00 | 391.14 | 7364 | 18417 |
| chromium_cdp | 3 | public_35 | warm | true | 365.13 | 0.00 | 365.13 | 27811 | 139967 |
| chromium_cdp | 3 | public_36 | warm | true | 572.70 | 0.00 | 572.70 | 22291 | 75775 |
| chromium_cdp | 3 | public_37 | warm | true | 2853.13 | 0.00 | 2853.13 | 8406 | 20740 |
| chromium_cdp | 3 | public_38 | warm | true | 6467.84 | 0.00 | 6467.84 | 23536 | 132518 |
| chromium_cdp | 3 | public_39 | warm | false | 264.28 | 0.00 | 264.28 | 745 | 28469 |
| chromium_cdp | 3 | public_40 | warm | true | 1990.67 | 0.00 | 1990.67 | 39823 | 595741 |
| chromium_cdp | 3 | public_41 | warm | true | 703.83 | 0.00 | 703.83 | 9812 | 258913 |
| chromium_cdp | 3 | public_42 | warm | true | 657.43 | 0.00 | 657.43 | 8991 | 531419 |
| chromium_cdp | 3 | public_43 | warm | true | 3728.84 | 0.00 | 3728.84 | 22718 | 310423 |
| chromium_cdp | 3 | public_44 | warm | true | 2711.58 | 0.00 | 2711.58 | 14381 | 64904 |
| chromium_cdp | 3 | public_45 | warm | true | 1902.56 | 0.00 | 1902.56 | 9891 | 12458 |
| chromium_cdp | 3 | public_46 | warm | true | 1808.44 | 0.00 | 1808.44 | 13351 | 61610 |
| chromium_cdp | 3 | public_47 | warm | true | 5809.33 | 0.00 | 5809.33 | 18722 | 695762 |
| chromium_cdp | 3 | public_48 | warm | true | 1705.83 | 0.00 | 1705.83 | 9664 | 400353 |
| chromium_cdp | 3 | public_49 | warm | true | 5554.52 | 0.00 | 5554.52 | 30318 | 330195 |
| chromium_cdp | 3 | public_50 | warm | true | 375.80 | 0.00 | 375.80 | 6144 | 15517 |
| chromium_cdp | 3 | public_51 | warm | true | 1170.54 | 0.00 | 1170.54 | 9525 | 19044 |
| chromium_cdp | 3 | public_52 | warm | false | 20000.26 | 0.00 | 20000.26 | 0 | 0 |
| chromium_cdp | 3 | public_53 | warm | false | 1366.09 | 0.00 | 1366.09 | 263 | 8620 |
| chromium_cdp | 3 | public_54 | warm | true | 1702.26 | 0.00 | 1702.26 | 19871 | 53531 |
| chromium_cdp | 3 | public_55 | warm | true | 2548.69 | 0.00 | 2548.69 | 24060 | 52874 |
| chromium_cdp | 3 | public_56 | warm | false | 20000.74 | 0.00 | 20000.74 | 0 | 0 |
| chromium_cdp | 3 | public_57 | warm | true | 1083.37 | 0.00 | 1083.37 | 12096 | 193151 |
| chromium_cdp | 3 | public_58 | warm | true | 900.58 | 0.00 | 900.58 | 1443 | 6242 |
| chromium_cdp | 3 | public_59 | warm | true | 146.65 | 0.00 | 146.65 | 2851 | 2358 |
| chromium_cdp | 3 | public_60 | warm | true | 731.69 | 0.00 | 731.69 | 926 | 646 |
| chromium_cdp | 3 | public_61 | warm | true | 191.76 | 0.00 | 191.76 | 21392 | 37786 |
| chromium_cdp | 3 | public_62 | warm | true | 120.10 | 0.00 | 120.10 | 21833 | 100773 |
| chromium_cdp | 3 | public_63 | warm | true | 614.68 | 0.00 | 614.68 | 29317 | 397301 |
| chromium_cdp | 3 | public_64 | warm | true | 1041.17 | 0.00 | 1041.17 | 7122 | 181526 |
| chromium_cdp | 3 | public_65 | warm | true | 1783.19 | 0.00 | 1783.19 | 4334 | 17828 |
| chromium_cdp | 3 | public_66 | warm | true | 285.59 | 0.00 | 285.59 | 28883 | 77867 |
| chromium_cdp | 3 | public_67 | warm | true | 3247.20 | 0.00 | 3247.20 | 9455 | 49905 |
| chromium_cdp | 3 | public_68 | warm | true | 2275.26 | 0.00 | 2275.26 | 11873 | 50416 |
| chromium_cdp | 3 | public_69 | warm | false | 20000.12 | 0.00 | 20000.12 | 0 | 0 |
| chromium_cdp | 3 | public_70 | warm | true | 4126.72 | 0.00 | 4126.72 | 26025 | 130613 |
| chromium_cdp | 3 | public_71 | warm | true | 4553.67 | 0.00 | 4553.67 | 11231 | 338125 |
| chromium_cdp | 3 | public_72 | warm | true | 1177.48 | 0.00 | 1177.48 | 11746 | 30377 |
| chromium_cdp | 3 | public_73 | warm | false | 20000.47 | 0.00 | 20000.47 | 0 | 0 |
| chromium_cdp | 3 | public_74 | warm | false | 20000.66 | 0.00 | 20000.66 | 228 | 8996 |
| chromium_cdp | 3 | public_75 | warm | true | 2023.98 | 0.00 | 2023.98 | 4374 | 7137 |
| chromium_cdp | 3 | public_76 | warm | true | 1844.57 | 0.00 | 1844.57 | 18285 | 80354 |
| chromium_cdp | 3 | public_77 | warm | true | 606.57 | 0.00 | 606.57 | 4814 | 5937 |
| chromium_cdp | 3 | public_78 | warm | true | 185.86 | 0.00 | 185.86 | 24551 | 333363 |
| chromium_cdp | 3 | public_79 | warm | true | 3252.71 | 0.00 | 3252.71 | 35064 | 1469650 |
| chromium_cdp | 3 | public_80 | warm | true | 3827.45 | 0.00 | 3827.45 | 37202 | 510630 |
| chromium_cdp | 3 | public_81 | warm | true | 1665.95 | 0.00 | 1665.95 | 31775 | 212435 |
| chromium_cdp | 3 | public_82 | warm | true | 1002.00 | 0.00 | 1002.00 | 31927 | 109978 |
| chromium_cdp | 3 | public_83 | warm | true | 2006.19 | 0.00 | 2006.19 | 58745 | 1087044 |
| chromium_cdp | 3 | public_84 | warm | true | 1876.21 | 0.00 | 1876.21 | 26906 | 557448 |
| chromium_cdp | 3 | public_85 | warm | true | 5864.08 | 0.00 | 5864.08 | 31957 | 1074575 |
| chromium_cdp | 3 | public_86 | warm | true | 1312.35 | 0.00 | 1312.35 | 29431 | 327341 |
| chromium_cdp | 3 | public_87 | warm | true | 1156.07 | 0.00 | 1156.07 | 41730 | 805379 |
| chromium_cdp | 3 | public_88 | warm | true | 1011.50 | 0.00 | 1011.50 | 41161 | 1441621 |
| chromium_cdp | 3 | public_89 | warm | true | 3414.43 | 0.00 | 3414.43 | 49609 | 2677763 |
| chromium_cdp | 3 | public_90 | warm | true | 18482.25 | 0.00 | 18482.25 | 32144 | 1695617 |
| chromium_cdp | 3 | public_91 | warm | true | 2110.76 | 0.00 | 2110.76 | 852 | 9483 |
| chromium_cdp | 3 | public_92 | warm | true | 74.53 | 0.00 | 74.53 | 787 | 723 |
| chromium_cdp | 3 | public_93 | warm | true | 3022.05 | 0.00 | 3022.05 | 18047 | 148819 |
| chromium_cdp | 3 | public_94 | warm | true | 591.69 | 0.00 | 591.69 | 1287 | 29375 |
| chromium_cdp | 3 | public_95 | warm | true | 1146.46 | 0.00 | 1146.46 | 20830 | 263739 |
| chromium_cdp | 3 | public_96 | warm | true | 3184.84 | 0.00 | 3184.84 | 15128 | 24828 |
| chromium_cdp | 3 | public_97 | warm | true | 1957.76 | 0.00 | 1957.76 | 15221 | 47257 |
| chromium_cdp | 3 | public_98 | warm | true | 3715.95 | 0.00 | 3715.95 | 19523 | 1823753 |
| chromium_cdp | 3 | public_99 | warm | true | 1307.87 | 0.00 | 1307.87 | 12451 | 261370 |
| chromium_cdp | 3 | public_100 | warm | true | 2817.61 | 0.00 | 2817.61 | 9800 | 35365 |
| chromium_cdp | 4 | public_1 | cold | true | 2079.12 | 384.98 | 2464.09 | 23816 | 316062 |
| chromium_cdp | 4 | public_2 | warm | true | 1336.60 | 0.00 | 1336.60 | 31834 | 2202682 |
| chromium_cdp | 4 | public_3 | warm | true | 1569.42 | 0.00 | 1569.42 | 34966 | 2014471 |
| chromium_cdp | 4 | public_4 | warm | true | 2068.62 | 0.00 | 2068.62 | 39013 | 3451096 |
| chromium_cdp | 4 | public_5 | warm | true | 1578.23 | 0.00 | 1578.23 | 34683 | 2290347 |
| chromium_cdp | 4 | public_6 | warm | true | 2118.88 | 0.00 | 2118.88 | 35075 | 2630531 |
| chromium_cdp | 4 | public_7 | warm | true | 2111.94 | 0.00 | 2111.94 | 55777 | 2039007 |
| chromium_cdp | 4 | public_8 | warm | true | 2140.93 | 0.00 | 2140.93 | 58494 | 3487732 |
| chromium_cdp | 4 | public_9 | warm | true | 1409.97 | 0.00 | 1409.97 | 31344 | 901599 |
| chromium_cdp | 4 | public_10 | warm | true | 1403.82 | 0.00 | 1403.82 | 32071 | 1362175 |
| chromium_cdp | 4 | public_11 | warm | true | 2607.24 | 0.00 | 2607.24 | 24942 | 328680 |
| chromium_cdp | 4 | public_12 | warm | true | 2860.29 | 0.00 | 2860.29 | 21176 | 186639 |
| chromium_cdp | 4 | public_13 | warm | true | 2671.41 | 0.00 | 2671.41 | 25375 | 240727 |
| chromium_cdp | 4 | public_14 | warm | true | 2853.19 | 0.00 | 2853.19 | 31029 | 607328 |
| chromium_cdp | 4 | public_15 | warm | true | 2315.44 | 0.00 | 2315.44 | 23210 | 212193 |
| chromium_cdp | 4 | public_16 | warm | false | 1506.11 | 0.00 | 1506.11 | 685 | 2115 |
| chromium_cdp | 4 | public_17 | warm | true | 2805.71 | 0.00 | 2805.71 | 24468 | 436512 |
| chromium_cdp | 4 | public_18 | warm | true | 3796.00 | 0.00 | 3796.00 | 22903 | 432576 |
| chromium_cdp | 4 | public_19 | warm | true | 2286.42 | 0.00 | 2286.42 | 26864 | 433455 |
| chromium_cdp | 4 | public_20 | warm | true | 2447.69 | 0.00 | 2447.69 | 23771 | 273726 |
| chromium_cdp | 4 | public_21 | warm | true | 2659.52 | 0.00 | 2659.52 | 17008 | 62708 |
| chromium_cdp | 4 | public_22 | warm | true | 737.26 | 0.00 | 737.26 | 10755 | 35843 |
| chromium_cdp | 4 | public_23 | warm | true | 2884.63 | 0.00 | 2884.63 | 17645 | 62661 |
| chromium_cdp | 4 | public_24 | warm | true | 1344.51 | 0.00 | 1344.51 | 21882 | 34168 |
| chromium_cdp | 4 | public_25 | warm | true | 241.76 | 0.00 | 241.76 | 24727 | 41700 |
| chromium_cdp | 4 | public_26 | warm | false | 576.93 | 0.00 | 576.93 | 782 | 28545 |
| chromium_cdp | 4 | public_27 | warm | false | 273.60 | 0.00 | 273.60 | 757 | 28481 |
| chromium_cdp | 4 | public_28 | warm | false | 604.36 | 0.00 | 604.36 | 765 | 28489 |
| chromium_cdp | 4 | public_29 | warm | true | 2846.84 | 0.00 | 2846.84 | 27230 | 245762 |
| chromium_cdp | 4 | public_30 | warm | true | 2536.22 | 0.00 | 2536.22 | 19643 | 611022 |
| chromium_cdp | 4 | public_31 | warm | true | 1334.03 | 0.00 | 1334.03 | 18255 | 301465 |
| chromium_cdp | 4 | public_32 | warm | true | 545.19 | 0.00 | 545.19 | 12112 | 24203 |
| chromium_cdp | 4 | public_33 | warm | true | 273.49 | 0.00 | 273.49 | 6375 | 17853 |
| chromium_cdp | 4 | public_34 | warm | true | 823.07 | 0.00 | 823.07 | 7364 | 18417 |
| chromium_cdp | 4 | public_35 | warm | true | 358.31 | 0.00 | 358.31 | 27811 | 139963 |
| chromium_cdp | 4 | public_36 | warm | true | 225.94 | 0.00 | 225.94 | 22291 | 75771 |
| chromium_cdp | 4 | public_37 | warm | true | 2771.95 | 0.00 | 2771.95 | 8406 | 20740 |
| chromium_cdp | 4 | public_38 | warm | true | 6486.75 | 0.00 | 6486.75 | 23536 | 132516 |
| chromium_cdp | 4 | public_39 | warm | false | 597.44 | 0.00 | 597.44 | 745 | 28469 |
| chromium_cdp | 4 | public_40 | warm | true | 2068.80 | 0.00 | 2068.80 | 39823 | 595743 |
| chromium_cdp | 4 | public_41 | warm | true | 384.96 | 0.00 | 384.96 | 9740 | 259091 |
| chromium_cdp | 4 | public_42 | warm | true | 654.80 | 0.00 | 654.80 | 8991 | 531462 |
| chromium_cdp | 4 | public_43 | warm | true | 7575.16 | 0.00 | 7575.16 | 23351 | 328509 |
| chromium_cdp | 4 | public_44 | warm | true | 2614.20 | 0.00 | 2614.20 | 14381 | 64904 |
| chromium_cdp | 4 | public_45 | warm | true | 1782.18 | 0.00 | 1782.18 | 9891 | 12458 |
| chromium_cdp | 4 | public_46 | warm | true | 1824.22 | 0.00 | 1824.22 | 13351 | 61610 |
| chromium_cdp | 4 | public_47 | warm | true | 5490.62 | 0.00 | 5490.62 | 18722 | 695283 |
| chromium_cdp | 4 | public_48 | warm | true | 2164.28 | 0.00 | 2164.28 | 9743 | 403709 |
| chromium_cdp | 4 | public_49 | warm | false | 20000.58 | 0.00 | 20000.58 | 0 | 0 |
| chromium_cdp | 4 | public_50 | warm | true | 422.09 | 0.00 | 422.09 | 6144 | 15517 |
| chromium_cdp | 4 | public_51 | warm | true | 148.29 | 0.00 | 148.29 | 9525 | 19044 |
| chromium_cdp | 4 | public_52 | warm | false | 20000.17 | 0.00 | 20000.17 | 0 | 0 |
| chromium_cdp | 4 | public_53 | warm | false | 1841.45 | 0.00 | 1841.45 | 263 | 8620 |
| chromium_cdp | 4 | public_54 | warm | true | 1523.48 | 0.00 | 1523.48 | 19871 | 53531 |
| chromium_cdp | 4 | public_55 | warm | true | 2421.78 | 0.00 | 2421.78 | 24060 | 52874 |
| chromium_cdp | 4 | public_56 | warm | false | 20000.93 | 0.00 | 20000.93 | 0 | 0 |
| chromium_cdp | 4 | public_57 | warm | true | 1342.15 | 0.00 | 1342.15 | 12096 | 193151 |
| chromium_cdp | 4 | public_58 | warm | true | 876.51 | 0.00 | 876.51 | 1443 | 6242 |
| chromium_cdp | 4 | public_59 | warm | true | 150.58 | 0.00 | 150.58 | 2851 | 2358 |
| chromium_cdp | 4 | public_60 | warm | true | 1208.11 | 0.00 | 1208.11 | 926 | 646 |
| chromium_cdp | 4 | public_61 | warm | true | 198.04 | 0.00 | 198.04 | 21392 | 37786 |
| chromium_cdp | 4 | public_62 | warm | true | 847.92 | 0.00 | 847.92 | 21833 | 100773 |
| chromium_cdp | 4 | public_63 | warm | true | 1013.55 | 0.00 | 1013.55 | 29317 | 397301 |
| chromium_cdp | 4 | public_64 | warm | true | 611.34 | 0.00 | 611.34 | 7122 | 181526 |
| chromium_cdp | 4 | public_65 | warm | true | 2078.13 | 0.00 | 2078.13 | 4334 | 17828 |
| chromium_cdp | 4 | public_66 | warm | true | 309.64 | 0.00 | 309.64 | 28883 | 77867 |
| chromium_cdp | 4 | public_67 | warm | true | 3853.62 | 0.00 | 3853.62 | 9455 | 49905 |
| chromium_cdp | 4 | public_68 | warm | true | 1356.42 | 0.00 | 1356.42 | 11873 | 50416 |
| chromium_cdp | 4 | public_69 | warm | false | 20000.57 | 0.00 | 20000.57 | 0 | 0 |
| chromium_cdp | 4 | public_70 | warm | true | 5432.99 | 0.00 | 5432.99 | 26025 | 130613 |
| chromium_cdp | 4 | public_71 | warm | true | 4617.03 | 0.00 | 4617.03 | 11231 | 338125 |
| chromium_cdp | 4 | public_72 | warm | true | 1180.45 | 0.00 | 1180.45 | 11752 | 30383 |
| chromium_cdp | 4 | public_73 | warm | false | 20000.28 | 0.00 | 20000.28 | 0 | 0 |
| chromium_cdp | 4 | public_74 | warm | false | 20000.14 | 0.00 | 20000.14 | 228 | 8996 |
| chromium_cdp | 4 | public_75 | warm | true | 1698.47 | 0.00 | 1698.47 | 4374 | 7137 |
| chromium_cdp | 4 | public_76 | warm | true | 1752.90 | 0.00 | 1752.90 | 18285 | 80354 |
| chromium_cdp | 4 | public_77 | warm | true | 150.28 | 0.00 | 150.28 | 4814 | 5937 |
| chromium_cdp | 4 | public_78 | warm | true | 349.37 | 0.00 | 349.37 | 24551 | 333363 |
| chromium_cdp | 4 | public_79 | warm | true | 3162.17 | 0.00 | 3162.17 | 35064 | 1469686 |
| chromium_cdp | 4 | public_80 | warm | true | 3603.38 | 0.00 | 3603.38 | 37202 | 510655 |
| chromium_cdp | 4 | public_81 | warm | true | 1820.48 | 0.00 | 1820.48 | 31510 | 212156 |
| chromium_cdp | 4 | public_82 | warm | true | 1050.72 | 0.00 | 1050.72 | 31927 | 109250 |
| chromium_cdp | 4 | public_83 | warm | true | 2035.35 | 0.00 | 2035.35 | 58745 | 1087043 |
| chromium_cdp | 4 | public_84 | warm | true | 2117.14 | 0.00 | 2117.14 | 26889 | 556832 |
| chromium_cdp | 4 | public_85 | warm | true | 2772.57 | 0.00 | 2772.57 | 31957 | 1061898 |
| chromium_cdp | 4 | public_86 | warm | true | 1819.86 | 0.00 | 1819.86 | 29431 | 327341 |
| chromium_cdp | 4 | public_87 | warm | true | 1650.27 | 0.00 | 1650.27 | 41776 | 805182 |
| chromium_cdp | 4 | public_88 | warm | true | 1125.41 | 0.00 | 1125.41 | 41099 | 1441118 |
| chromium_cdp | 4 | public_89 | warm | true | 3650.00 | 0.00 | 3650.00 | 49572 | 2676829 |
| chromium_cdp | 4 | public_90 | warm | true | 17159.98 | 0.00 | 17159.98 | 32144 | 1695634 |
| chromium_cdp | 4 | public_91 | warm | true | 2149.03 | 0.00 | 2149.03 | 852 | 9483 |
| chromium_cdp | 4 | public_92 | warm | true | 217.82 | 0.00 | 217.82 | 787 | 723 |
| chromium_cdp | 4 | public_93 | warm | true | 3037.50 | 0.00 | 3037.50 | 17778 | 148633 |
| chromium_cdp | 4 | public_94 | warm | true | 639.36 | 0.00 | 639.36 | 1287 | 29375 |
| chromium_cdp | 4 | public_95 | warm | true | 1142.03 | 0.00 | 1142.03 | 20830 | 263739 |
| chromium_cdp | 4 | public_96 | warm | true | 2823.30 | 0.00 | 2823.30 | 15128 | 24828 |
| chromium_cdp | 4 | public_97 | warm | true | 2032.36 | 0.00 | 2032.36 | 15221 | 47257 |
| chromium_cdp | 4 | public_98 | warm | true | 2559.62 | 0.00 | 2559.62 | 19523 | 1808413 |
| chromium_cdp | 4 | public_99 | warm | true | 1892.70 | 0.00 | 1892.70 | 12451 | 261388 |
| chromium_cdp | 4 | public_100 | warm | true | 3856.54 | 0.00 | 3856.54 | 9800 | 35365 |
| ai_go | 4 | public_1 | cold | true | 1542.45 | 0.00 | 1542.45 | 29611 | 253942 |
| ai_go | 4 | public_2 | warm | true | 2287.86 | 0.00 | 2287.86 | 33849 | 2017450 |
| ai_go | 4 | public_3 | warm | true | 2503.26 | 0.00 | 2503.26 | 34992 | 1876166 |
| ai_go | 4 | public_4 | warm | true | 2480.32 | 0.00 | 2480.32 | 37329 | 2873942 |
| ai_go | 4 | public_5 | warm | true | 1996.49 | 0.00 | 1996.49 | 36305 | 2075994 |
| ai_go | 4 | public_6 | warm | true | 1835.23 | 0.00 | 1835.23 | 34924 | 2113993 |
| ai_go | 4 | public_7 | warm | true | 1776.53 | 0.00 | 1776.53 | 38034 | 1866457 |
| ai_go | 4 | public_8 | warm | true | 1990.01 | 0.00 | 1990.01 | 41138 | 3249113 |
| ai_go | 4 | public_9 | warm | true | 1673.88 | 0.00 | 1673.88 | 34305 | 864585 |
| ai_go | 4 | public_10 | warm | true | 1737.91 | 0.00 | 1737.91 | 33604 | 1282423 |
| ai_go | 4 | public_11 | warm | true | 1190.60 | 0.00 | 1190.60 | 32245 | 254179 |
| ai_go | 4 | public_12 | warm | true | 1054.13 | 0.00 | 1054.13 | 27573 | 102830 |
| ai_go | 4 | public_13 | warm | true | 1410.97 | 0.00 | 1410.97 | 28417 | 174324 |
| ai_go | 4 | public_14 | warm | true | 1246.07 | 0.00 | 1246.07 | 37086 | 127227 |
| ai_go | 4 | public_15 | warm | true | 1405.79 | 0.00 | 1405.79 | 27983 | 156046 |
| ai_go | 4 | public_16 | warm | true | 1370.73 | 0.00 | 1370.73 | 31502 | 132573 |
| ai_go | 4 | public_17 | warm | true | 1166.73 | 0.00 | 1166.73 | 28243 | 95737 |
| ai_go | 4 | public_18 | warm | true | 1411.55 | 0.00 | 1411.55 | 28722 | 135080 |
| ai_go | 4 | public_19 | warm | true | 1427.33 | 0.00 | 1427.33 | 28738 | 91459 |
| ai_go | 4 | public_20 | warm | true | 1650.34 | 0.00 | 1650.34 | 30696 | 213319 |
| ai_go | 4 | public_21 | warm | true | 719.50 | 0.00 | 719.50 | 20425 | 64188 |
| ai_go | 4 | public_22 | warm | true | 328.04 | 0.00 | 328.04 | 11811 | 34115 |
| ai_go | 4 | public_23 | warm | true | 1146.09 | 0.00 | 1146.09 | 21057 | 64188 |
| ai_go | 4 | public_24 | warm | true | 594.57 | 0.00 | 594.57 | 24613 | 34115 |
| ai_go | 4 | public_25 | warm | true | 193.94 | 0.00 | 193.94 | 28544 | 41024 |
| ai_go | 4 | public_26 | warm | false | 20000.07 | 0.00 | 20000.07 | 317 | 5653 |
| ai_go | 4 | public_27 | warm | false | 20000.33 | 0.00 | 20000.33 | 313 | 5649 |
| ai_go | 4 | public_28 | warm | false | 20000.95 | 0.00 | 20000.95 | 315 | 5651 |
| ai_go | 4 | public_29 | warm | false | 20000.83 | 0.00 | 20000.83 | 313 | 5649 |
| ai_go | 4 | public_30 | warm | true | 2247.85 | 0.00 | 2247.85 | 28864 | 608683 |
| ai_go | 4 | public_31 | warm | true | 641.30 | 0.00 | 641.30 | 27325 | 300785 |
| ai_go | 4 | public_32 | warm | true | 554.35 | 0.00 | 554.35 | 14226 | 19625 |
| ai_go | 4 | public_33 | warm | true | 281.69 | 0.00 | 281.69 | 7461 | 19717 |
| ai_go | 4 | public_34 | warm | true | 1274.98 | 0.00 | 1274.98 | 9550 | 18594 |
| ai_go | 4 | public_35 | warm | true | 276.68 | 0.00 | 276.68 | 32345 | 222341 |
| ai_go | 4 | public_36 | warm | true | 406.32 | 0.00 | 406.32 | 24988 | 121721 |
| ai_go | 4 | public_37 | warm | true | 1111.95 | 0.00 | 1111.95 | 9882 | 19190 |
| ai_go | 4 | public_38 | warm | true | 1001.45 | 0.00 | 1001.45 | 19678 | 132697 |
| ai_go | 4 | public_39 | warm | false | 20000.87 | 0.00 | 20000.87 | 310 | 5646 |
| ai_go | 4 | public_40 | warm | true | 119.53 | 0.00 | 119.53 | 34229 | 372981 |
| ai_go | 4 | public_41 | warm | true | 61.80 | 0.00 | 61.80 | 11937 | 258828 |
| ai_go | 4 | public_42 | warm | true | 456.11 | 0.00 | 456.11 | 13755 | 531745 |
| ai_go | 4 | public_43 | warm | true | 2626.67 | 0.00 | 2626.67 | 26858 | 289647 |
| ai_go | 4 | public_44 | warm | true | 1475.24 | 0.00 | 1475.24 | 17076 | 59911 |
| ai_go | 4 | public_45 | warm | true | 244.58 | 0.00 | 244.58 | 11297 | 12617 |
| ai_go | 4 | public_46 | warm | true | 44.71 | 0.00 | 44.71 | 15530 | 41608 |
| ai_go | 4 | public_47 | warm | true | 1072.69 | 0.00 | 1072.69 | 21407 | 500979 |
| ai_go | 4 | public_48 | warm | true | 952.10 | 0.00 | 952.10 | 10775 | 185401 |
| ai_go | 4 | public_49 | warm | true | 795.78 | 0.00 | 795.78 | 44520 | 209845 |
| ai_go | 4 | public_50 | warm | true | 348.75 | 0.00 | 348.75 | 8333 | 15864 |
| ai_go | 4 | public_51 | warm | true | 468.44 | 0.00 | 468.44 | 10866 | 19150 |
| ai_go | 4 | public_52 | warm | false | 10000.51 | 0.00 | 10000.51 | 0 | 0 |
| ai_go | 4 | public_53 | warm | false | 20000.83 | 0.00 | 20000.83 | 306 | 8293 |
| ai_go | 4 | public_54 | warm | true | 78.97 | 0.00 | 78.97 | 21504 | 51923 |
| ai_go | 4 | public_55 | warm | true | 943.63 | 0.00 | 943.63 | 27669 | 53921 |
| ai_go | 4 | public_56 | warm | true | 346.47 | 0.00 | 346.47 | 339 | 1145 |
| ai_go | 4 | public_57 | warm | true | 231.96 | 0.00 | 231.96 | 14484 | 192265 |
| ai_go | 4 | public_58 | warm | true | 1656.38 | 0.00 | 1656.38 | 1906 | 6688 |
| ai_go | 4 | public_59 | warm | true | 221.89 | 0.00 | 221.89 | 1701 | 2394 |
| ai_go | 4 | public_60 | warm | true | 790.29 | 0.00 | 790.29 | 1078 | 755 |
| ai_go | 4 | public_61 | warm | true | 285.04 | 0.00 | 285.04 | 24849 | 44341 |
| ai_go | 4 | public_62 | warm | true | 591.41 | 0.00 | 591.41 | 28325 | 117674 |
| ai_go | 4 | public_63 | warm | true | 1266.65 | 0.00 | 1266.65 | 46205 | 432718 |
| ai_go | 4 | public_64 | warm | true | 805.89 | 0.00 | 805.89 | 8857 | 183808 |
| ai_go | 4 | public_65 | warm | true | 2056.44 | 0.00 | 2056.44 | 4952 | 19304 |
| ai_go | 4 | public_66 | warm | true | 401.14 | 0.00 | 401.14 | 11000 | 83835 |
| ai_go | 4 | public_67 | warm | true | 3350.33 | 0.00 | 3350.33 | 12300 | 52338 |
| ai_go | 4 | public_68 | warm | true | 1642.52 | 0.00 | 1642.52 | 13505 | 52894 |
| ai_go | 4 | public_69 | warm | false | 20000.02 | 0.00 | 20000.02 | 0 | 0 |
| ai_go | 4 | public_70 | warm | true | 4675.81 | 0.00 | 4675.81 | 24452 | 147152 |
| ai_go | 4 | public_71 | warm | true | 5168.44 | 0.00 | 5168.44 | 15893 | 344807 |
| ai_go | 4 | public_72 | warm | true | 1260.20 | 0.00 | 1260.20 | 13330 | 33087 |
| ai_go | 4 | public_73 | warm | false | 20006.35 | 0.00 | 20006.35 | 0 | 0 |
| ai_go | 4 | public_74 | warm | true | 1348.09 | 0.00 | 1348.09 | 2739 | 77046 |
| ai_go | 4 | public_75 | warm | true | 1993.02 | 0.00 | 1993.02 | 5396 | 8237 |
| ai_go | 4 | public_76 | warm | true | 1983.89 | 0.00 | 1983.89 | 21716 | 100312 |
| ai_go | 4 | public_77 | warm | true | 361.36 | 0.00 | 361.36 | 5259 | 6559 |
| ai_go | 4 | public_78 | warm | true | 452.57 | 0.00 | 452.57 | 26733 | 335733 |
| ai_go | 4 | public_79 | warm | true | 5987.96 | 0.00 | 5987.96 | 60481 | 1492088 |
| ai_go | 4 | public_80 | warm | true | 3877.02 | 0.00 | 3877.02 | 46226 | 521013 |
| ai_go | 4 | public_81 | warm | true | 1613.46 | 0.00 | 1613.46 | 35712 | 223099 |
| ai_go | 4 | public_82 | warm | true | 1253.27 | 0.00 | 1253.27 | 34687 | 115627 |
| ai_go | 4 | public_83 | warm | true | 2981.68 | 0.00 | 2981.68 | 69767 | 1010779 |
| ai_go | 4 | public_84 | warm | true | 1971.53 | 0.00 | 1971.53 | 28996 | 564548 |
| ai_go | 4 | public_85 | warm | true | 5589.24 | 0.00 | 5589.24 | 42185 | 1115509 |
| ai_go | 4 | public_86 | warm | true | 1859.41 | 0.00 | 1859.41 | 42586 | 335222 |
| ai_go | 4 | public_87 | warm | true | 6084.20 | 0.00 | 6084.20 | 60604 | 614876 |
| ai_go | 4 | public_88 | warm | true | 1400.95 | 0.00 | 1400.95 | 58291 | 1455892 |
| ai_go | 4 | public_89 | warm | true | 4431.83 | 0.00 | 4431.83 | 69333 | 1789409 |
| ai_go | 4 | public_90 | warm | true | 16884.50 | 0.00 | 16884.50 | 36993 | 1703963 |
| ai_go | 4 | public_91 | warm | true | 2207.85 | 0.00 | 2207.85 | 937 | 9603 |
| ai_go | 4 | public_92 | warm | true | 254.81 | 0.00 | 254.81 | 905 | 738 |
| ai_go | 4 | public_93 | warm | true | 3206.18 | 0.00 | 3206.18 | 22169 | 149512 |
| ai_go | 4 | public_94 | warm | true | 1332.44 | 0.00 | 1332.44 | 3043 | 31237 |
| ai_go | 4 | public_95 | warm | true | 1710.46 | 0.00 | 1710.46 | 22780 | 276107 |
| ai_go | 4 | public_96 | warm | true | 2417.77 | 0.00 | 2417.77 | 17366 | 28321 |
| ai_go | 4 | public_97 | warm | true | 1894.24 | 0.00 | 1894.24 | 14194 | 51468 |
| ai_go | 4 | public_98 | warm | true | 3443.75 | 0.00 | 3443.75 | 16358 | 1824622 |
| ai_go | 4 | public_99 | warm | true | 2134.58 | 0.00 | 2134.58 | 13549 | 265318 |
| ai_go | 4 | public_100 | warm | true | 6148.81 | 0.00 | 6148.81 | 11024 | 37267 |
| ai_go | 5 | public_1 | cold | true | 1258.53 | 0.00 | 1258.53 | 29611 | 254154 |
| ai_go | 5 | public_2 | warm | true | 2448.11 | 0.00 | 2448.11 | 33849 | 2017662 |
| ai_go | 5 | public_3 | warm | true | 2003.87 | 0.00 | 2003.87 | 34992 | 1876378 |
| ai_go | 5 | public_4 | warm | true | 2010.37 | 0.00 | 2010.37 | 37329 | 2874094 |
| ai_go | 5 | public_5 | warm | true | 2075.85 | 0.00 | 2075.85 | 36305 | 2076258 |
| ai_go | 5 | public_6 | warm | true | 1969.69 | 0.00 | 1969.69 | 34924 | 2114205 |
| ai_go | 5 | public_7 | warm | true | 2544.67 | 0.00 | 2544.67 | 38034 | 1866669 |
| ai_go | 5 | public_8 | warm | true | 2003.34 | 0.00 | 2003.34 | 41138 | 3249113 |
| ai_go | 5 | public_9 | warm | true | 1514.80 | 0.00 | 1514.80 | 34305 | 864585 |
| ai_go | 5 | public_10 | warm | true | 2040.10 | 0.00 | 2040.10 | 33604 | 1282635 |
| ai_go | 5 | public_11 | warm | true | 1195.55 | 0.00 | 1195.55 | 32245 | 254179 |
| ai_go | 5 | public_12 | warm | true | 1030.58 | 0.00 | 1030.58 | 27573 | 102830 |
| ai_go | 5 | public_13 | warm | true | 1413.03 | 0.00 | 1413.03 | 28417 | 174324 |
| ai_go | 5 | public_14 | warm | true | 947.98 | 0.00 | 947.98 | 37086 | 127227 |
| ai_go | 5 | public_15 | warm | true | 1244.40 | 0.00 | 1244.40 | 27983 | 156046 |
| ai_go | 5 | public_16 | warm | true | 1520.41 | 0.00 | 1520.41 | 31502 | 132573 |
| ai_go | 5 | public_17 | warm | true | 1213.57 | 0.00 | 1213.57 | 28243 | 95737 |
| ai_go | 5 | public_18 | warm | true | 1413.11 | 0.00 | 1413.11 | 28722 | 135080 |
| ai_go | 5 | public_19 | warm | true | 1333.94 | 0.00 | 1333.94 | 28738 | 91459 |
| ai_go | 5 | public_20 | warm | true | 1353.94 | 0.00 | 1353.94 | 30696 | 213319 |
| ai_go | 5 | public_21 | warm | true | 695.36 | 0.00 | 695.36 | 20425 | 64188 |
| ai_go | 5 | public_22 | warm | true | 252.85 | 0.00 | 252.85 | 11811 | 34115 |
| ai_go | 5 | public_23 | warm | true | 1582.37 | 0.00 | 1582.37 | 21057 | 64188 |
| ai_go | 5 | public_24 | warm | true | 597.26 | 0.00 | 597.26 | 24613 | 34115 |
| ai_go | 5 | public_25 | warm | true | 195.34 | 0.00 | 195.34 | 28614 | 40831 |
| ai_go | 5 | public_26 | warm | false | 20001.02 | 0.00 | 20001.02 | 317 | 5653 |
| ai_go | 5 | public_27 | warm | false | 20000.46 | 0.00 | 20000.46 | 313 | 5649 |
| ai_go | 5 | public_28 | warm | false | 20000.62 | 0.00 | 20000.62 | 315 | 5651 |
| ai_go | 5 | public_29 | warm | false | 20000.31 | 0.00 | 20000.31 | 313 | 5649 |
| ai_go | 5 | public_30 | warm | true | 2142.49 | 0.00 | 2142.49 | 28864 | 608683 |
| ai_go | 5 | public_31 | warm | true | 647.69 | 0.00 | 647.69 | 27325 | 300785 |
| ai_go | 5 | public_32 | warm | true | 349.72 | 0.00 | 349.72 | 14226 | 19625 |
| ai_go | 5 | public_33 | warm | true | 427.68 | 0.00 | 427.68 | 7797 | 19964 |
| ai_go | 5 | public_34 | warm | true | 710.64 | 0.00 | 710.64 | 9550 | 18594 |
| ai_go | 5 | public_35 | warm | true | 69.62 | 0.00 | 69.62 | 32345 | 222341 |
| ai_go | 5 | public_36 | warm | true | 33.64 | 0.00 | 33.64 | 24988 | 121721 |
| ai_go | 5 | public_37 | warm | true | 1071.67 | 0.00 | 1071.67 | 9882 | 19190 |
| ai_go | 5 | public_38 | warm | true | 1073.08 | 0.00 | 1073.08 | 19678 | 132701 |
| ai_go | 5 | public_39 | warm | false | 20000.03 | 0.00 | 20000.03 | 310 | 5646 |
| ai_go | 5 | public_40 | warm | true | 114.54 | 0.00 | 114.54 | 34229 | 372981 |
| ai_go | 5 | public_41 | warm | true | 67.07 | 0.00 | 67.07 | 11937 | 258828 |
| ai_go | 5 | public_42 | warm | true | 437.46 | 0.00 | 437.46 | 13755 | 531775 |
| ai_go | 5 | public_43 | warm | true | 2901.41 | 0.00 | 2901.41 | 26627 | 286124 |
| ai_go | 5 | public_44 | warm | true | 1184.35 | 0.00 | 1184.35 | 17076 | 59911 |
| ai_go | 5 | public_45 | warm | true | 385.39 | 0.00 | 385.39 | 11297 | 12617 |
| ai_go | 5 | public_46 | warm | true | 48.67 | 0.00 | 48.67 | 15530 | 41608 |
| ai_go | 5 | public_47 | warm | true | 1109.89 | 0.00 | 1109.89 | 21407 | 500979 |
| ai_go | 5 | public_48 | warm | true | 955.42 | 0.00 | 955.42 | 10775 | 185401 |
| ai_go | 5 | public_49 | warm | true | 796.07 | 0.00 | 796.07 | 44520 | 209845 |
| ai_go | 5 | public_50 | warm | true | 789.45 | 0.00 | 789.45 | 8333 | 15864 |
| ai_go | 5 | public_51 | warm | true | 347.00 | 0.00 | 347.00 | 10866 | 19150 |
| ai_go | 5 | public_52 | warm | false | 10000.47 | 0.00 | 10000.47 | 0 | 0 |
| ai_go | 5 | public_53 | warm | false | 20000.60 | 0.00 | 20000.60 | 306 | 8293 |
| ai_go | 5 | public_54 | warm | false | 10000.73 | 0.00 | 10000.73 | 0 | 0 |
| ai_go | 5 | public_55 | warm | true | 944.88 | 0.00 | 944.88 | 27669 | 53921 |
| ai_go | 5 | public_56 | warm | false | 10000.61 | 0.00 | 10000.61 | 0 | 0 |
| ai_go | 5 | public_57 | warm | true | 256.38 | 0.00 | 256.38 | 14484 | 192265 |
| ai_go | 5 | public_58 | warm | true | 1410.53 | 0.00 | 1410.53 | 1906 | 6688 |
| ai_go | 5 | public_59 | warm | true | 205.76 | 0.00 | 205.76 | 1701 | 2394 |
| ai_go | 5 | public_60 | warm | true | 1089.36 | 0.00 | 1089.36 | 1078 | 755 |
| ai_go | 5 | public_61 | warm | true | 283.82 | 0.00 | 283.82 | 24849 | 44341 |
| ai_go | 5 | public_62 | warm | true | 1241.10 | 0.00 | 1241.10 | 28325 | 117674 |
| ai_go | 5 | public_63 | warm | true | 1222.77 | 0.00 | 1222.77 | 46205 | 432718 |
| ai_go | 5 | public_64 | warm | true | 865.47 | 0.00 | 865.47 | 8857 | 183808 |
| ai_go | 5 | public_65 | warm | true | 2156.33 | 0.00 | 2156.33 | 4952 | 19304 |
| ai_go | 5 | public_66 | warm | true | 383.99 | 0.00 | 383.99 | 11000 | 83835 |
| ai_go | 5 | public_67 | warm | true | 3518.29 | 0.00 | 3518.29 | 12300 | 52336 |
| ai_go | 5 | public_68 | warm | true | 1975.01 | 0.00 | 1975.01 | 13505 | 52892 |
| ai_go | 5 | public_69 | warm | false | 20000.19 | 0.00 | 20000.19 | 0 | 0 |
| ai_go | 5 | public_70 | warm | true | 3458.81 | 0.00 | 3458.81 | 24452 | 147152 |
| ai_go | 5 | public_71 | warm | true | 4868.82 | 0.00 | 4868.82 | 15893 | 344947 |
| ai_go | 5 | public_72 | warm | true | 1256.76 | 0.00 | 1256.76 | 13330 | 33087 |
| ai_go | 5 | public_73 | warm | false | 20006.67 | 0.00 | 20006.67 | 0 | 0 |
| ai_go | 5 | public_74 | warm | true | 1379.30 | 0.00 | 1379.30 | 2739 | 77046 |
| ai_go | 5 | public_75 | warm | true | 2057.92 | 0.00 | 2057.92 | 5396 | 8237 |
| ai_go | 5 | public_76 | warm | true | 1767.35 | 0.00 | 1767.35 | 21716 | 100312 |
| ai_go | 5 | public_77 | warm | true | 611.79 | 0.00 | 611.79 | 5259 | 6559 |
| ai_go | 5 | public_78 | warm | true | 534.81 | 0.00 | 534.81 | 26733 | 335733 |
| ai_go | 5 | public_79 | warm | true | 6131.98 | 0.00 | 6131.98 | 60481 | 1491935 |
| ai_go | 5 | public_80 | warm | true | 3528.58 | 0.00 | 3528.58 | 46226 | 521038 |
| ai_go | 5 | public_81 | warm | true | 2484.16 | 0.00 | 2484.16 | 35457 | 222346 |
| ai_go | 5 | public_82 | warm | true | 1293.55 | 0.00 | 1293.55 | 34687 | 115627 |
| ai_go | 5 | public_83 | warm | true | 3471.49 | 0.00 | 3471.49 | 69767 | 1010779 |
| ai_go | 5 | public_84 | warm | true | 2339.09 | 0.00 | 2339.09 | 28956 | 564583 |
| ai_go | 5 | public_85 | warm | true | 10685.87 | 0.00 | 10685.87 | 42270 | 1116436 |
| ai_go | 5 | public_86 | warm | true | 1933.11 | 0.00 | 1933.11 | 42590 | 335199 |
| ai_go | 5 | public_87 | warm | true | 6757.67 | 0.00 | 6757.67 | 60604 | 614876 |
| ai_go | 5 | public_88 | warm | true | 1354.78 | 0.00 | 1354.78 | 58291 | 1455725 |
| ai_go | 5 | public_89 | warm | true | 4676.73 | 0.00 | 4676.73 | 69545 | 1788324 |
| ai_go | 5 | public_90 | warm | true | 17475.19 | 0.00 | 17475.19 | 36993 | 1704052 |
| ai_go | 5 | public_91 | warm | true | 2292.40 | 0.00 | 2292.40 | 937 | 9603 |
| ai_go | 5 | public_92 | warm | true | 253.06 | 0.00 | 253.06 | 905 | 738 |
| ai_go | 5 | public_93 | warm | true | 3254.28 | 0.00 | 3254.28 | 22169 | 149512 |
| ai_go | 5 | public_94 | warm | true | 1311.92 | 0.00 | 1311.92 | 3043 | 31237 |
| ai_go | 5 | public_95 | warm | true | 1427.66 | 0.00 | 1427.66 | 22780 | 276107 |
| ai_go | 5 | public_96 | warm | true | 2471.18 | 0.00 | 2471.18 | 17366 | 28321 |
| ai_go | 5 | public_97 | warm | true | 2044.26 | 0.00 | 2044.26 | 14196 | 51470 |
| ai_go | 5 | public_98 | warm | true | 3589.20 | 0.00 | 3589.20 | 16358 | 1824379 |
| ai_go | 5 | public_99 | warm | true | 2471.76 | 0.00 | 2471.76 | 13549 | 265318 |
| ai_go | 5 | public_100 | warm | true | 6257.75 | 0.00 | 6257.75 | 11024 | 37267 |
| chromium_cdp | 5 | public_1 | cold | true | 1964.06 | 384.90 | 2348.96 | 23816 | 316251 |
| chromium_cdp | 5 | public_2 | warm | true | 1588.29 | 0.00 | 1588.29 | 31834 | 2202682 |
| chromium_cdp | 5 | public_3 | warm | true | 1600.38 | 0.00 | 1600.38 | 34966 | 2073471 |
| chromium_cdp | 5 | public_4 | warm | true | 2308.16 | 0.00 | 2308.16 | 39013 | 3451096 |
| chromium_cdp | 5 | public_5 | warm | true | 1278.52 | 0.00 | 1278.52 | 34683 | 2217883 |
| chromium_cdp | 5 | public_6 | warm | true | 1303.16 | 0.00 | 1303.16 | 33596 | 2277895 |
| chromium_cdp | 5 | public_7 | warm | true | 2067.19 | 0.00 | 2067.19 | 55777 | 2039007 |
| chromium_cdp | 5 | public_8 | warm | true | 1887.17 | 0.00 | 1887.17 | 58494 | 3733888 |
| chromium_cdp | 5 | public_9 | warm | true | 1604.83 | 0.00 | 1604.83 | 31344 | 901599 |
| chromium_cdp | 5 | public_10 | warm | true | 1189.31 | 0.00 | 1189.31 | 32071 | 1362175 |
| chromium_cdp | 5 | public_11 | warm | true | 2426.40 | 0.00 | 2426.40 | 24942 | 254245 |
| chromium_cdp | 5 | public_12 | warm | true | 2854.91 | 0.00 | 2854.91 | 21176 | 186639 |
| chromium_cdp | 5 | public_13 | warm | true | 2513.46 | 0.00 | 2513.46 | 25375 | 240727 |
| chromium_cdp | 5 | public_14 | warm | true | 2735.64 | 0.00 | 2735.64 | 31029 | 607328 |
| chromium_cdp | 5 | public_15 | warm | true | 2419.65 | 0.00 | 2419.65 | 23210 | 212193 |
| chromium_cdp | 5 | public_16 | warm | true | 2019.42 | 0.00 | 2019.42 | 25279 | 134914 |
| chromium_cdp | 5 | public_17 | warm | true | 2357.95 | 0.00 | 2357.95 | 23266 | 153297 |
| chromium_cdp | 5 | public_18 | warm | true | 3775.02 | 0.00 | 3775.02 | 22903 | 432573 |
| chromium_cdp | 5 | public_19 | warm | true | 1794.50 | 0.00 | 1794.50 | 25704 | 149456 |
| chromium_cdp | 5 | public_20 | warm | true | 2451.28 | 0.00 | 2451.28 | 23771 | 273726 |
| chromium_cdp | 5 | public_21 | warm | true | 2796.77 | 0.00 | 2796.77 | 17008 | 62708 |
| chromium_cdp | 5 | public_22 | warm | true | 853.14 | 0.00 | 853.14 | 10761 | 35843 |
| chromium_cdp | 5 | public_23 | warm | true | 3253.30 | 0.00 | 3253.30 | 17645 | 62661 |
| chromium_cdp | 5 | public_24 | warm | true | 1116.53 | 0.00 | 1116.53 | 21840 | 34121 |
| chromium_cdp | 5 | public_25 | warm | true | 254.78 | 0.00 | 254.78 | 25082 | 41534 |
| chromium_cdp | 5 | public_26 | warm | false | 723.07 | 0.00 | 723.07 | 782 | 28545 |
| chromium_cdp | 5 | public_27 | warm | false | 423.37 | 0.00 | 423.37 | 757 | 28481 |
| chromium_cdp | 5 | public_28 | warm | false | 545.89 | 0.00 | 545.89 | 765 | 28489 |
| chromium_cdp | 5 | public_29 | warm | true | 2805.35 | 0.00 | 2805.35 | 27499 | 245946 |
| chromium_cdp | 5 | public_30 | warm | true | 2624.38 | 0.00 | 2624.38 | 19643 | 611026 |
| chromium_cdp | 5 | public_31 | warm | true | 1444.15 | 0.00 | 1444.15 | 18255 | 301465 |
| chromium_cdp | 5 | public_32 | warm | true | 528.58 | 0.00 | 528.58 | 12112 | 24203 |
| chromium_cdp | 5 | public_33 | warm | true | 323.47 | 0.00 | 323.47 | 7009 | 18281 |
| chromium_cdp | 5 | public_34 | warm | true | 1310.60 | 0.00 | 1310.60 | 7364 | 18417 |
| chromium_cdp | 5 | public_35 | warm | true | 641.14 | 0.00 | 641.14 | 27811 | 139963 |
| chromium_cdp | 5 | public_36 | warm | true | 248.12 | 0.00 | 248.12 | 22291 | 75777 |
| chromium_cdp | 5 | public_37 | warm | true | 2786.99 | 0.00 | 2786.99 | 8406 | 20740 |
| chromium_cdp | 5 | public_38 | warm | true | 6649.68 | 0.00 | 6649.68 | 23536 | 132514 |
| chromium_cdp | 5 | public_39 | warm | false | 587.29 | 0.00 | 587.29 | 745 | 28469 |
| chromium_cdp | 5 | public_40 | warm | true | 2034.07 | 0.00 | 2034.07 | 39823 | 595741 |
| chromium_cdp | 5 | public_41 | warm | true | 369.04 | 0.00 | 369.04 | 9740 | 259091 |
| chromium_cdp | 5 | public_42 | warm | true | 393.78 | 0.00 | 393.78 | 8991 | 531404 |
| chromium_cdp | 5 | public_43 | warm | true | 4203.69 | 0.00 | 4203.69 | 23351 | 328493 |
| chromium_cdp | 5 | public_44 | warm | true | 2619.35 | 0.00 | 2619.35 | 14381 | 64904 |
| chromium_cdp | 5 | public_45 | warm | true | 2278.33 | 0.00 | 2278.33 | 9891 | 12458 |
| chromium_cdp | 5 | public_46 | warm | true | 1915.64 | 0.00 | 1915.64 | 13351 | 61610 |
| chromium_cdp | 5 | public_47 | warm | true | 5469.21 | 0.00 | 5469.21 | 18722 | 694707 |
| chromium_cdp | 5 | public_48 | warm | true | 2159.96 | 0.00 | 2159.96 | 9743 | 403729 |
| chromium_cdp | 5 | public_49 | warm | true | 2980.57 | 0.00 | 2980.57 | 30286 | 330086 |
| chromium_cdp | 5 | public_50 | warm | true | 601.18 | 0.00 | 601.18 | 6144 | 15517 |
| chromium_cdp | 5 | public_51 | warm | true | 840.24 | 0.00 | 840.24 | 9525 | 19044 |
| chromium_cdp | 5 | public_52 | warm | false | 20000.47 | 0.00 | 20000.47 | 0 | 0 |
| chromium_cdp | 5 | public_53 | warm | false | 2457.39 | 0.00 | 2457.39 | 263 | 8620 |
| chromium_cdp | 5 | public_54 | warm | true | 1551.33 | 0.00 | 1551.33 | 19871 | 53531 |
| chromium_cdp | 5 | public_55 | warm | true | 2315.69 | 0.00 | 2315.69 | 24060 | 52874 |
| chromium_cdp | 5 | public_56 | warm | false | 20000.95 | 0.00 | 20000.95 | 0 | 0 |
| chromium_cdp | 5 | public_57 | warm | true | 1623.65 | 0.00 | 1623.65 | 12096 | 193151 |
| chromium_cdp | 5 | public_58 | warm | true | 844.42 | 0.00 | 844.42 | 1443 | 6242 |
| chromium_cdp | 5 | public_59 | warm | true | 206.54 | 0.00 | 206.54 | 2851 | 2358 |
| chromium_cdp | 5 | public_60 | warm | true | 1176.69 | 0.00 | 1176.69 | 926 | 646 |
| chromium_cdp | 5 | public_61 | warm | true | 196.90 | 0.00 | 196.90 | 21392 | 37786 |
| chromium_cdp | 5 | public_62 | warm | true | 1177.67 | 0.00 | 1177.67 | 21833 | 100773 |
| chromium_cdp | 5 | public_63 | warm | true | 809.17 | 0.00 | 809.17 | 29317 | 397301 |
| chromium_cdp | 5 | public_64 | warm | true | 791.07 | 0.00 | 791.07 | 7122 | 181526 |
| chromium_cdp | 5 | public_65 | warm | true | 2377.73 | 0.00 | 2377.73 | 4334 | 17828 |
| chromium_cdp | 5 | public_66 | warm | true | 321.28 | 0.00 | 321.28 | 28883 | 77867 |
| chromium_cdp | 5 | public_67 | warm | true | 3609.32 | 0.00 | 3609.32 | 9455 | 49907 |
| chromium_cdp | 5 | public_68 | warm | true | 1908.81 | 0.00 | 1908.81 | 11873 | 50418 |
| chromium_cdp | 5 | public_69 | warm | false | 20000.29 | 0.00 | 20000.29 | 0 | 0 |
| chromium_cdp | 5 | public_70 | warm | true | 4203.20 | 0.00 | 4203.20 | 26025 | 130613 |
| chromium_cdp | 5 | public_71 | warm | true | 5032.63 | 0.00 | 5032.63 | 11231 | 338125 |
| chromium_cdp | 5 | public_72 | warm | true | 1186.17 | 0.00 | 1186.17 | 11752 | 30383 |
| chromium_cdp | 5 | public_73 | warm | false | 20000.26 | 0.00 | 20000.26 | 0 | 0 |
| chromium_cdp | 5 | public_74 | warm | false | 20000.44 | 0.00 | 20000.44 | 228 | 8996 |
| chromium_cdp | 5 | public_75 | warm | true | 2123.88 | 0.00 | 2123.88 | 4374 | 7137 |
| chromium_cdp | 5 | public_76 | warm | true | 1627.04 | 0.00 | 1627.04 | 18285 | 80354 |
| chromium_cdp | 5 | public_77 | warm | true | 1442.72 | 0.00 | 1442.72 | 4814 | 5937 |
| chromium_cdp | 5 | public_78 | warm | true | 401.93 | 0.00 | 401.93 | 24570 | 333388 |
| chromium_cdp | 5 | public_79 | warm | true | 3239.83 | 0.00 | 3239.83 | 35064 | 1469686 |
| chromium_cdp | 5 | public_80 | warm | true | 3759.17 | 0.00 | 3759.17 | 37202 | 510635 |
| chromium_cdp | 5 | public_81 | warm | true | 2068.23 | 0.00 | 2068.23 | 31588 | 212136 |
| chromium_cdp | 5 | public_82 | warm | true | 1149.73 | 0.00 | 1149.73 | 31927 | 109978 |
| chromium_cdp | 5 | public_83 | warm | true | 2087.90 | 0.00 | 2087.90 | 58745 | 1087043 |
| chromium_cdp | 5 | public_84 | warm | true | 1905.46 | 0.00 | 1905.46 | 26850 | 557874 |
| chromium_cdp | 5 | public_85 | warm | true | 2985.95 | 0.00 | 2985.95 | 31937 | 1062277 |
| chromium_cdp | 5 | public_86 | warm | true | 1577.41 | 0.00 | 1577.41 | 29429 | 327312 |
| chromium_cdp | 5 | public_87 | warm | true | 1474.49 | 0.00 | 1474.49 | 41776 | 805182 |
| chromium_cdp | 5 | public_88 | warm | true | 1035.35 | 0.00 | 1035.35 | 41104 | 1441739 |
| chromium_cdp | 5 | public_89 | warm | true | 4275.44 | 0.00 | 4275.44 | 49729 | 2681343 |
| chromium_cdp | 5 | public_90 | warm | true | 18039.12 | 0.00 | 18039.12 | 32144 | 1695560 |
| chromium_cdp | 5 | public_91 | warm | true | 2113.78 | 0.00 | 2113.78 | 852 | 9483 |
| chromium_cdp | 5 | public_92 | warm | true | 203.40 | 0.00 | 203.40 | 787 | 723 |
| chromium_cdp | 5 | public_93 | warm | true | 3059.10 | 0.00 | 3059.10 | 18272 | 147578 |
| chromium_cdp | 5 | public_94 | warm | true | 610.24 | 0.00 | 610.24 | 1287 | 29375 |
| chromium_cdp | 5 | public_95 | warm | true | 1030.71 | 0.00 | 1030.71 | 20830 | 263739 |
| chromium_cdp | 5 | public_96 | warm | true | 2361.10 | 0.00 | 2361.10 | 15128 | 24828 |
| chromium_cdp | 5 | public_97 | warm | true | 1841.49 | 0.00 | 1841.49 | 15221 | 47256 |
| chromium_cdp | 5 | public_98 | warm | true | 2464.06 | 0.00 | 2464.06 | 19015 | 1803816 |
| chromium_cdp | 5 | public_99 | warm | false | 11206.73 | 0.00 | 11206.73 | 0 | 0 |
| chromium_cdp | 5 | public_100 | warm | true | 5877.39 | 0.00 | 5877.39 | 9800 | 35365 |

## Errors

- ai_go, run 1, public_26: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 1, public_27: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 1, public_28: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 1, public_29: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 1, public_39: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 1, public_53: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 1, public_69: open: live navigation: context deadline exceeded
- ai_go, run 1, public_73: open: live navigation: context deadline exceeded
- ai_go, run 1, public_89: public smoke returned HTTP 403
- chromium_cdp, run 1, public_26: public smoke returned HTTP 403
- chromium_cdp, run 1, public_27: public smoke returned HTTP 403
- chromium_cdp, run 1, public_28: public smoke returned HTTP 403
- chromium_cdp, run 1, public_29: public smoke returned HTTP 403
- chromium_cdp, run 1, public_39: public smoke returned HTTP 403
- chromium_cdp, run 1, public_52: context deadline exceeded
- chromium_cdp, run 1, public_53: public smoke returned HTTP 403
- chromium_cdp, run 1, public_56: navigation readiness: context deadline exceeded
- chromium_cdp, run 1, public_73: context deadline exceeded
- chromium_cdp, run 1, public_74: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- chromium_cdp, run 2, public_26: public smoke returned HTTP 403
- chromium_cdp, run 2, public_27: public smoke returned HTTP 403
- chromium_cdp, run 2, public_28: public smoke returned HTTP 403
- chromium_cdp, run 2, public_29: public smoke returned HTTP 403
- chromium_cdp, run 2, public_39: public smoke returned HTTP 403
- chromium_cdp, run 2, public_52: context deadline exceeded
- chromium_cdp, run 2, public_53: public smoke returned HTTP 403
- chromium_cdp, run 2, public_56: navigation readiness: context deadline exceeded
- chromium_cdp, run 2, public_69: navigation readiness: context deadline exceeded
- chromium_cdp, run 2, public_73: context deadline exceeded
- chromium_cdp, run 2, public_74: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- chromium_cdp, run 2, public_87: navigation readiness: context deadline exceeded
- ai_go, run 2, public_26: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 2, public_27: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 2, public_28: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 2, public_29: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 2, public_39: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 2, public_52: open: Get "https://www.gnu.org/": dial tcp 209.51.188.116:443: i/o timeout
- ai_go, run 2, public_53: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 2, public_55: open: Get "https://www.php.net/": net/http: TLS handshake timeout
- ai_go, run 2, public_69: open: live navigation: context deadline exceeded
- ai_go, run 2, public_73: open: context deadline exceeded
- ai_go, run 2, public_96: open: live navigation failed: net::ERR_CONNECTION_RESET
- ai_go, run 3, public_26: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 3, public_27: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 3, public_28: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 3, public_29: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 3, public_39: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 3, public_52: open: Get "https://www.gnu.org/": dial tcp 209.51.188.116:443: i/o timeout
- ai_go, run 3, public_53: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 3, public_73: open: context deadline exceeded
- chromium_cdp, run 3, public_16: public smoke returned HTTP 429
- chromium_cdp, run 3, public_26: public smoke returned HTTP 403
- chromium_cdp, run 3, public_27: public smoke returned HTTP 403
- chromium_cdp, run 3, public_28: public smoke returned HTTP 403
- chromium_cdp, run 3, public_39: public smoke returned HTTP 403
- chromium_cdp, run 3, public_52: context deadline exceeded
- chromium_cdp, run 3, public_53: public smoke returned HTTP 403
- chromium_cdp, run 3, public_56: navigation readiness: context deadline exceeded
- chromium_cdp, run 3, public_69: navigation readiness: context deadline exceeded
- chromium_cdp, run 3, public_73: context deadline exceeded
- chromium_cdp, run 3, public_74: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- chromium_cdp, run 4, public_16: public smoke returned HTTP 429
- chromium_cdp, run 4, public_26: public smoke returned HTTP 403
- chromium_cdp, run 4, public_27: public smoke returned HTTP 403
- chromium_cdp, run 4, public_28: public smoke returned HTTP 403
- chromium_cdp, run 4, public_39: public smoke returned HTTP 403
- chromium_cdp, run 4, public_49: navigation readiness: context deadline exceeded
- chromium_cdp, run 4, public_52: context deadline exceeded
- chromium_cdp, run 4, public_53: public smoke returned HTTP 403
- chromium_cdp, run 4, public_56: navigation readiness: context deadline exceeded
- chromium_cdp, run 4, public_69: navigation readiness: context deadline exceeded
- chromium_cdp, run 4, public_73: context deadline exceeded
- chromium_cdp, run 4, public_74: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 4, public_26: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 4, public_27: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 4, public_28: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 4, public_29: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 4, public_39: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 4, public_52: open: Get "https://www.gnu.org/": dial tcp 209.51.188.116:443: i/o timeout
- ai_go, run 4, public_53: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 4, public_69: open: live navigation: context deadline exceeded
- ai_go, run 4, public_73: open: context deadline exceeded
- ai_go, run 5, public_26: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 5, public_27: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 5, public_28: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 5, public_29: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 5, public_39: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 5, public_52: open: Get "https://www.gnu.org/": dial tcp 209.51.188.116:443: i/o timeout
- ai_go, run 5, public_53: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- ai_go, run 5, public_54: open: Get "https://www.python.org/": dial tcp: lookup www.python.org: i/o timeout
- ai_go, run 5, public_56: open: Get "https://www.ruby-lang.org/": dial tcp: lookup www.ruby-lang.org: i/o timeout
- ai_go, run 5, public_69: open: live navigation: context deadline exceeded
- ai_go, run 5, public_73: open: context deadline exceeded
- chromium_cdp, run 5, public_26: public smoke returned HTTP 403
- chromium_cdp, run 5, public_27: public smoke returned HTTP 403
- chromium_cdp, run 5, public_28: public smoke returned HTTP 403
- chromium_cdp, run 5, public_39: public smoke returned HTTP 403
- chromium_cdp, run 5, public_52: context deadline exceeded
- chromium_cdp, run 5, public_53: public smoke returned HTTP 403
- chromium_cdp, run 5, public_56: navigation readiness: context deadline exceeded
- chromium_cdp, run 5, public_69: navigation readiness: context deadline exceeded
- chromium_cdp, run 5, public_73: context deadline exceeded
- chromium_cdp, run 5, public_74: assertion failed: visible text must contain "" (last snapshot error: <nil>): context deadline exceeded
- chromium_cdp, run 5, public_99: navigate https://graphql.org/: net::ERR_NAME_NOT_RESOLVED

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
