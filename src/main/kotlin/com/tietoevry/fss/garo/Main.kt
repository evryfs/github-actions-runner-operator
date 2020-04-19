package com.tietoevry.fss.garo

import com.fasterxml.jackson.databind.ObjectMapper
import com.fasterxml.jackson.jaxrs.json.JacksonJaxbJsonProvider
import com.fasterxml.jackson.module.kotlin.registerKotlinModule
import io.fabric8.kubernetes.client.DefaultKubernetesClient
import org.apache.cxf.feature.LoggingFeature
import javax.ws.rs.client.ClientBuilder

fun main(args: Array<String>) {
    DefaultKubernetesClient().use {
        val objectMapper = ObjectMapper().registerKotlinModule()
        val jaxbJsonProvider = JacksonJaxbJsonProvider(objectMapper, JacksonJaxbJsonProvider.DEFAULT_ANNOTATIONS)
        val clientBuilder = ClientBuilder.newBuilder()
                .register(LoggingFeature())
                .register(jaxbJsonProvider)
                .build()

        val controller = ActionRunnerController(clientBuilder, it)
        controller.controlLoop()
    }
}

class Main {
}