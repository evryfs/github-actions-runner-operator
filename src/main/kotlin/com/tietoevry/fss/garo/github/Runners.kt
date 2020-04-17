package com.tietoevry.fss.garo.github

import com.fasterxml.jackson.annotation.JsonProperty

data class Runners(@JsonProperty("total_count") val totalCount: Int,
                   val runners: Collection<Runner> )