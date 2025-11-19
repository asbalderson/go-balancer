package handlers

/*
we either need a type for each response which impelments the ServeHTTP(w ResponseWriter, r *Request) or we need to write a function for each response which uses the HandleFunc(patther string, handler func(as serve http)) that does stuff

we need to handle /status for the pod status that returns the hostname, ip address, service, and timestamp (and count) since it returns count we can create the object before hand and then every time we call it we can increment the counter. but there must be a good way to update the timestamp at this time, we'll figure that out.

errors should return what went wrong and where it was

we will have to write the headers for every handler, to http.ResponseWriter.Header().Set("Content-Type", "application/json") and write the status of the response (which are builtins in http, we can look them up) and then encode the data into the response json.NewEncoder(http.ResponseWritter).Encode(data) which will let us return json data with a proper response.

to me it makes sense to have a "return json response" method that does all that stuff we mentioned above, and we just tell it to return a json response with the data, and error code, and it will pass forward the response from the paragraph ahead. that should be the simplest way to do it.

I think it makes sense to have a type for each response we want to handle. and i will have to play with the timestamp being updated on each call to the endpoint, but we can work that out, since all those things we want to return may update, maybe we just pass in an empty struct and look them up each time?

we can get the pod ip address and hostname from the environment variables which will be exposed from the k8s config, no problem there, we can set the manually for testing locally. for count then, we should pass in a variable and then increment it each time and then create the struct and return it. nice and easy it also solves the timestamp problem!
*/
