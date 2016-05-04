Multi-Factor Risk Service
=================================================================================================================================================================================================

The *multifactorriskservice* project provides a prototype risk service server for the [Intervention Engine](https://github.com/intervention-engine/ie) project. The *multifactorriskservice* server interfaces with a [REDCap](http://projectredcap.org/) database to import recorded risk scores for patients, based on a multi-factor risk model.  The *multifactorriskservice* server also provides risk component data in a format that allows the Intervention Engine [frontend](https://github.com/intervention-engine/frontend) to properly draw the "risk pies".

In addition to the REDCap data importer, the *multifactorriskservice* provides a _mock_ implementation for generating synthetic risk scores to allow testing and development without a REDCap server.

License
-------

Copyright 2016 The MITRE Corporation

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
