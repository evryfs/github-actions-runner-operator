package com.tietoevry.fss.garo

import com.fasterxml.jackson.databind.ObjectMapper
import com.fasterxml.jackson.jaxrs.json.JacksonJaxbJsonProvider
import com.fasterxml.jackson.module.kotlin.registerKotlinModule
import com.tietoevry.fss.garo.github.RunnerApi
import io.fabric8.kubernetes.client.DefaultKubernetesClient
import org.apache.cxf.feature.LoggingFeature
import org.apache.cxf.jaxrs.client.JAXRSClientFactoryBean

fun main(args: Array<String>) {
    DefaultKubernetesClient().use {
        val factoryBean = JAXRSClientFactoryBean()
        val objectMapper = ObjectMapper().registerKotlinModule()
        val jaxbJsonProvider = JacksonJaxbJsonProvider(objectMapper, JacksonJaxbJsonProvider.DEFAULT_ANNOTATIONS)
        factoryBean.features = listOf(LoggingFeature())
        factoryBean.serviceClass = RunnerApi::class.java
        factoryBean.address = "https://api.github.com"
        factoryBean.providers = listOf(jaxbJsonProvider)

        val controller = ActionRunnerController(factoryBean.create(RunnerApi::class.java), it)
        controller.controlLoop()
    }
}

class Main {
}