package com.tietoevry.fss.garo

import com.fasterxml.jackson.databind.ObjectMapper
import com.fasterxml.jackson.module.kotlin.registerKotlinModule
import io.quarkus.jackson.ObjectMapperCustomizer
import javax.inject.Singleton

@Singleton
class JacksonConf : ObjectMapperCustomizer {

    override fun customize(objectMapper: ObjectMapper) {
        objectMapper.registerKotlinModule()
    }
}