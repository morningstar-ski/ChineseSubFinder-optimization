@{
    route_policies = @(
        @{
            route_key = "movie.native_chinese"
            keep_count = 2
            required = $true
        },
        @{
            route_key = "movie.subtitlecat_translated"
            keep_count = 2
            required = $true
        },
        @{
            route_key = "movie.english_fallback"
            keep_count = 2
            required = $true
        },
        @{
            route_key = "movie.safe_fail"
            keep_count = 1
            required = $true
        },
        @{
            route_key = "movie.llm_fallback"
            keep_count = 1
            required = $true
        },
        @{
            route_key = "series.native_chinese"
            keep_count = 2
            required = $true
        },
        @{
            route_key = "series.subtitlecat_translated"
            keep_count = 2
            required = $true
        },
        @{
            route_key = "series.english_fallback"
            keep_count = 2
            required = $true
        },
        @{
            route_key = "series.safe_fail"
            keep_count = 1
            required = $false
        },
        @{
            route_key = "series.llm_fallback"
            keep_count = 1
            required = $true
        }
    )

    full = @(
        @{
            name = "prepare-static-build-start-movie-native"
            sample_spec = "the-nice-guys-2016.json"
            run_static_checks = $true
            build_image = $true
            start_container = $true
            run_e2e_matrix = $true
            primary_chinese_suppliers = @("subhd")
            expected_route_key = "movie.native_chinese"
        },
        @{
            name = "movie-translated-explicit"
            sample_spec = "interstellar-2014.json"
            run_e2e_matrix = $true
            primary_chinese_suppliers = @("__none__")
            english_fallback_suppliers = @("subtitlecat")
            enable_subtitlecat_translated_chinese_fallback = $true
            expected_route_key = "movie.subtitlecat_translated"
        },
        @{
            name = "movie-english-fallback-default-chain"
            sample_spec = "interstellar-2014.json"
            run_e2e_matrix = $true
            primary_chinese_suppliers = @("__none__")
            english_fallback_suppliers = @("subdl", "subtitlecat", "moviesubtitles")
            expected_route_key = "movie.english_fallback"
            expected_winning_supplier = "subdl"
        },
        @{
            name = "movie-safe-fail-guard"
            sample_spec = "the-hours-2002.json"
            run_e2e_matrix = $true
            primary_chinese_suppliers = @("__none__")
            english_fallback_suppliers = @("subtitlecat")
            accept_no_sub_found = $true
            expected_route_key = "movie.safe_fail"
        },
        @{
            name = "series-default-real-library"
            sample_spec = "the-boys-s01e03.json"
            run_e2e_matrix = $true
            primary_chinese_suppliers = @("subhd")
            english_fallback_suppliers = @("subtitlecat")
            expected_route_key = "series.native_chinese"
        },
        @{
            name = "series-english-fallback-default-chain"
            sample_spec = "crowded-room-s01e02.json"
            run_e2e_matrix = $true
            primary_chinese_suppliers = @("__none__")
            english_fallback_suppliers = @("subdl", "subtitlecat")
            expected_route_key = "series.english_fallback"
            expected_winning_supplier = "subdl"
        },
        @{
            name = "series-translated-explicit"
            sample_spec = "george-lopez-s01e01.json"
            run_e2e_matrix = $true
            primary_chinese_suppliers = @("__none__")
            english_fallback_suppliers = @("subtitlecat")
            enable_subtitlecat_translated_chinese_fallback = $true
            expected_route_key = "series.subtitlecat_translated"
        },
        @{
            name = "movie-llm-fallback"
            sample_spec = "interstellar-2014.json"
            run_e2e_matrix = $true
            primary_chinese_suppliers = @("__none__")
            english_fallback_suppliers = @("subtitlecat")
            enable_llm_fallback = $true
            requires_llm = $true
            expected_route_key = "movie.llm_fallback"
        },
        @{
            name = "series-llm-fallback"
            sample_spec = "crowded-room-s01e03.json"
            run_e2e_matrix = $true
            primary_chinese_suppliers = @("__none__")
            english_fallback_suppliers = @("subtitlecat")
            enable_llm_fallback = $true
            requires_llm = $true
            expected_route_key = "series.llm_fallback"
        }
    )

    expanded = @(
        @{
            name = "movie-english-fallback-alt"
            sample_spec = "memento-sample.json"
            start_container = $true
            run_e2e_matrix = $true
            primary_chinese_suppliers = @("__none__")
            english_fallback_suppliers = @("subtitlecat")
            expected_route_key = "movie.english_fallback"
        },
        @{
            name = "series-english-fallback-alt"
            sample_spec = "george-lopez-s01e01.json"
            run_e2e_matrix = $true
            primary_chinese_suppliers = @("__none__")
            english_fallback_suppliers = @("subtitlecat")
            expected_route_key = "series.english_fallback"
        },
        @{
            name = "series-native-alt"
            sample_spec = "the-boys-s01e02.json"
            run_e2e_matrix = $true
            primary_chinese_suppliers = @("subhd")
            english_fallback_suppliers = @("subtitlecat")
            expected_route_key = "series.native_chinese"
        }
    )

    llm_only = @(
        @{
            name = "movie-llm-fallback"
            sample_spec = "interstellar-2014.json"
            start_container = $true
            run_e2e_matrix = $true
            primary_chinese_suppliers = @("__none__")
            english_fallback_suppliers = @("subtitlecat")
            enable_llm_fallback = $true
            requires_llm = $true
            expected_route_key = "movie.llm_fallback"
        },
        @{
            name = "series-llm-fallback"
            sample_spec = "crowded-room-s01e03.json"
            run_e2e_matrix = $true
            primary_chinese_suppliers = @("__none__")
            english_fallback_suppliers = @("subtitlecat")
            enable_llm_fallback = $true
            requires_llm = $true
            expected_route_key = "series.llm_fallback"
        }
    )
}
