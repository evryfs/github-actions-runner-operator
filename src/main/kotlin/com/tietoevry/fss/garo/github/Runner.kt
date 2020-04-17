package com.tietoevry.fss.garo.github

data class Runner(val id: Int,
                  val name: String,
                  val os: String,
                  val status: String) {

    fun isOnline() = status == "online"
    fun isOffline() = status == "offline"
}

