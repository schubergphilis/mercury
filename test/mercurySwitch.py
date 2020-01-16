import requests
from urllib3.exceptions import InsecureRequestWarning
import json
import random
import sys
from concurrent.futures import ThreadPoolExecutor
requests.packages.urllib3.disable_warnings(category=InsecureRequestWarning)

class Mercury:
  
  def __init__(self,env,datacenter,bearer):
    self.env = env
    self.datacenter = datacenter
    self.convention = {
      'odd': 'center1',
      'even': 'center2'
    } 
    self.datacenters = ['center1','center2']
    self.bearer = bearer
    self.headers = {'Authorization': bearer}
    self.envs = {
      'testing' : {
        'ip': 'localhost', 
        'prefix': '',
        'datacenter': ['center1']      
      },
      'acceptance': {
        'ip': 'number', 
        'prefix': 'a',
        'datacenter': ['center2']
      },
      'production': {
        'ip': 'number', 
        'prefix': 'p',
        'datacenter': ['center1','center2']
      }
    }
    
    try:
      self.url = 'https://{}:9001/api/v1/healthchecks'.format(self.envs[env]['ip'])
    except KeyError as e:
      self.help()

  def parity_filter(self,nodeName):
    if nodeName.startswith(self.envs[self.env]['prefix']) and (self.datacenter in self.envs[self.env]['datacenter']):
      print(nodeName)
      if (nodeName == ""):
          return False
      elif (int(nodeName[-1]) % 2 == 0) and (self.convention['even'] == self.datacenter): 
        return True
      elif (int(nodeName[-1]) % 2 != 0) and (self.convention['odd'] == self.datacenter): 
        return True
      else:
        return False
    else:
      return False
    

  def get_workers(self):
    response = requests.get(url=self.url,headers=self.headers,verify=False)
    data = json.loads(response.json()['data'])
    return data['workers']


  def get_hosts_uuids(self,hostSet):
    pass
    
  def help(self):
    print('Argument list:')
    print('1st: choose one datacenter to shutdown from this list: {}'.format(self.datacenters))
    print('2nd: choose one environment to shutdown from this list: {}'.format(list(self.envs.keys())))
    print('Example: python3 mercurySwitch.py center1 testing')
    exit(1)


if __name__ == '__main__':

  # To get the token, login on mercury, inspect the connection network properties and look for the Authorization header
  bearer = 'BEARER eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJleHBpcmUiOjE1NzkxODA0NDUsInVzZXJuYW1lIjoidGVzdCJ9.77Zrm1A8lsrDKCk2hYrNIJFqZp2ghVc4MW_u_rS1aLG9wCCXPH28GYIJ4GnFkc88yhAg8AMrhhu1CKY7KPbAuw'

  try:
    sideToKill = sys.argv[1]
    env = sys.argv[2]
  except Exception as e:
    print(e)
    print('You must pass the datacenter and the environment')

  mercury = Mercury(env,sideToKill,bearer)
  workers = mercury.get_workers() 
  hosts = []
  

  for worker in workers:
    uuid = worker['uuid']
    nodeName = worker['nodename'].split('_')[0]
    print(nodeName)
    node = (nodeName,uuid)

    if mercury.parity_filter(nodeName):
      hosts.append(node)

  actionList = ["online","offline","maintenance","automatic"]
  urlList = []
  for host in hosts:
    urlList.append("{}/admin/{}/status/{}".format(mercury.url,host[1],random.choice(actionList)))

  with ThreadPoolExecutor(8) as executor:
    futures = executor.map(lambda r: requests.post(r,verify=False,headers=mercury.headers), urlList)
    #futures = executor.map(lambda r: print(r, urlList))
  
  for future in futures:
    print(future)
